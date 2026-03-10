# ApplyWise — Backend

> Resume ATS analyzer powered by Go + Flutter. Uses AI to score ATS compatibility, rewrite bullet points, estimate salary ranges, and optimize LinkedIn profiles.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Flutter App                            │
│               (iOS & Android — RevenueCat IAP)              │
└───────────────────────┬─────────────────────────────────────┘
                        │ HTTPS
                        ▼
┌─────────────────────────────────────────────────────────────┐
│               AWS API Gateway v2 (HTTP API)                 │
│                  catch-all → Lambda proxy                   │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│            API Lambda  (Go · Chi v5 router)                 │
│                                                             │
│  Auth  ──►  AWS Cognito (email + JWT, custom SES emails)    │
│  Data  ──►  DynamoDB  (single-table design, PK/SK + 2 GSI)  │
│  Files ──►  S3  (AES-256, versioned, no public access)      │
│  Email ──►  AWS SES v2  (domain: hirefy.careers)            │
│  Pay   ──►  Stripe  (web checkout + webhooks)               │
│  Pay   ──►  RevenueCat  (App Store + Play Store webhooks)   │
│  Push  ──►  Firebase FCM  (iOS + Android notifications)     │
│  Jobs  ──►  SQS queue  (async optimization tasks)           │
└─────────────────────────────────────────────────────────────┘
                        │ SQS event trigger (batch=1)
                        ▼
┌─────────────────────────────────────────────────────────────┐
│           Worker Lambda  (Go · 60s timeout)                 │
│                                                             │
│  Consumes OptimizationJobMessage from SQS                   │
│  ──►  OpenAI API  (resume parse, rewrite, salary estimate,  │
│                    LinkedIn optimization)                   │
│  ──►  DynamoDB  (writes result, updates job status)         │
│  ──►  FCM  (push notification to user on completion)        │
└─────────────────────────────────────────────────────────────┘
```

---

## Async Optimization Flow

```
Flutter app                 API Lambda              Worker Lambda
    │                           │                        │
    │──POST /resumes/optimize──►│                        │
    │                           │─── enqueue SQS ───────►│
    │◄── 202 Accepted + jobID ──│                        │
    │                           │             [processes with OpenAI]
    │──GET /optimize/jobs/{id}──►│                        │
    │◄── { status: "queued" } ──│                        │
    │                           │         [writes result to DynamoDB]
    │                           │         [sends FCM push notification]
    │◄──── PUSH NOTIFICATION ───────────────────────────►│
    │──GET /optimize/jobs/{id}──►│                        │
    │◄──{ status: "completed" }─│                        │
```

The client polls every 3 seconds **and** receives an FCM push notification on completion.

---

## Code Structure — Hexagonal / Ports & Adapters

```
cmd/
 ├── api/main.go        ← wires everything, runs as Lambda or plain HTTP (local)
 └── worker/main.go     ← SQS consumer Lambda

internal/
 ├── core/
 │   ├── domain/        ← pure Go structs (User, Resume, Subscription,
 │   │                     OptimizationJob, CreditTransaction…)
 │   └── ports/
 │       ├── inbound/   ← service interfaces (AuthService, ResumeOptimizerService…)
 │       └── outbound/  ← repository + adapter interfaces (AIService, PaymentGateway…)
 │
 ├── application/
 │   └── service/       ← business logic, depends only on port interfaces
 │
 └── adapters/
     ├── inbound/
     │   └── http/      ← Chi handlers + JWT auth middleware
     └── outbound/
         ├── ai/openai/            ← OpenAI adapter
         ├── auth/cognito/         ← AWS Cognito adapter + JWKS JWT verifier
         ├── email/                ← AWS SES v2 adapter + HTML templates
         ├── notification/fcm/     ← Firebase FCM adapter
         ├── payment/stripe/       ← Stripe SDK adapter + webhook verifier
         ├── persistence/dynamodb/ ← 6 DynamoDB repositories (single table)
         └── queue/sqs/            ← SQS publisher
```

---

## Subscription & Credit System

Users can subscribe through two independent billing paths:

```
Web (browser / Stripe)                Mobile (iOS & Android)
        │                                      │
        ▼                                      ▼
Stripe Checkout Session             RevenueCat IAP aggregator
        │                                      │
        ▼                                      ▼
 Stripe webhook ──────────────────► RevenueCat webhook
        │                                      │
        └──────────────┬────────────────────────┘
                       ▼
             DynamoDB Subscription record
             { store: "stripe" | "app_store" | "play_store" }
             { plan: "free" | "basic" | "premium" }
             { credits: N }

Credit ledger: every add/use event is a CreditTransaction row
```

---

## Infrastructure (Terraform Modules)

| Module | What it provisions |
|---|---|
| `dynamodb` | Single table `applywise-prod`, PAY_PER_REQUEST, PK+SK, GSI1+GSI2 |
| `cognito` | User pool, app client, custom SES email sender |
| `lambda` (×2) | API Lambda + Worker Lambda (`provided.al2` — Go binary) |
| `sqs` | Standard queue for async optimization jobs |
| `api_gateway` | HTTP API v2, catch-all route → Lambda proxy, auto-deploy |
| `s3` | Resume bucket — AES-256, versioned, zero public access |
| `budget` | Monthly spend alert at 80% / 100% of $10 |

---

## Tech Stack

| Layer | Technology |
|---|---|
| Mobile | Flutter (iOS + Android) |
| Backend language | Go 1.24 |
| HTTP router | Chi v5 |
| Compute | AWS Lambda (`provided.al2`) |
| API entry point | AWS API Gateway v2 |
| Database | AWS DynamoDB (single-table design) |
| Object storage | AWS S3 |
| Authentication | AWS Cognito + JWT (JWKS) |
| Email | AWS SES v2 |
| Async queue | AWS SQS |
| AI / LLM | OpenAI API |
| Push notifications | Firebase Cloud Messaging |
| Payments (web) | Stripe |
| Payments (mobile) | RevenueCat |
| IaC | Terraform |

---

## API Endpoints

### Public

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Health check |
| `POST` | `/api/v1/auth/signup` | Register a new user |
| `POST` | `/api/v1/auth/signin` | Sign in |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `POST` | `/api/v1/auth/confirm` | Confirm account (email code) |
| `POST` | `/api/v1/auth/resend-code` | Resend confirmation code |
| `POST` | `/api/v1/auth/forgot-password` | Request password reset |
| `POST` | `/api/v1/auth/confirm-forgot-password` | Confirm password reset |
| `POST` | `/api/v1/webhooks/stripe` | Stripe webhook receiver |
| `POST` | `/api/v1/webhooks/revenuecat` | RevenueCat webhook receiver |

### Protected (requires `Authorization: Bearer <token>`)

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/resumes` | Upload a resume |
| `GET` | `/api/v1/resumes` | List resumes |
| `POST` | `/api/v1/resumes/manual` | Create a manual resume |
| `GET/PUT` | `/api/v1/resumes/manual/{resumeID}` | Get / update manual resume |
| `DELETE` | `/api/v1/resumes/{resumeID}` | Delete a resume |
| `POST` | `/api/v1/resumes/optimize` | Start async ATS optimization |
| `GET` | `/api/v1/resumes/optimize/jobs/{jobID}` | Poll optimization job status |
| `GET` | `/api/v1/resumes/optimized` | List optimized resumes |
| `GET/PUT` | `/api/v1/resumes/optimized/{optimizedID}` | Get / update optimized resume |
| `POST` | `/api/v1/resumes/linkedin/optimize` | Optimize LinkedIn profile |
| `GET/POST/DELETE` | `/api/v1/subscription` | Subscription management |
| `POST` | `/api/v1/subscription/checkout` | Create Stripe checkout session |
| `GET` | `/api/v1/subscription/credits` | Get credit transaction history |
| `GET` | `/api/v1/users/me` | Get current user profile |
| `POST` | `/api/v1/users/me/fcm-token` | Register FCM push token |
| `DELETE` | `/api/v1/users/me` | Delete account |

---

## Environment Variables

| Variable | Description |
|---|---|
| `DYNAMODB_TABLE` | DynamoDB single-table name |
| `COGNITO_USER_POOL_ID` | Cognito user pool ID |
| `COGNITO_CLIENT_ID` | Cognito app client ID |
| `STRIPE_SECRET_KEY` | Stripe API key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook HMAC secret |
| `OPENAI_API_KEY` | OpenAI API key |
| `OPTIMIZATION_QUEUE_URL` | SQS queue URL |
| `FIREBASE_CREDENTIALS_FILE` | Path to Firebase service account JSON |
| `FIREBASE_PROJECT_ID` | Firebase project ID |
| `SES_FROM_EMAIL` | Sender email address (SES) |
| `REVENUECAT_WEBHOOK_SECRET` | RevenueCat webhook secret |
| `ENVIRONMENT` | `dev` or `prod` |
| `PORT` | Local HTTP port (default: `8080`) |

---

## Running Locally

```bash
# Install dependencies
make deps

# Run the API server locally (plain HTTP on :8080)
make run-local

# Run tests
make test

# Lint
make lint
```

## Deployment

```bash
# Build Go binaries + zip + terraform apply (dev)
make deploy

# Build only
make build

# Deploy Lambda functions directly (prod)
make deploy-aws
```
