# ApplyWise Backend

Backend service for ApplyWise - AI-powered resume optimization platform.

## Architecture

- **Clean Architecture** with Ports and Adapters (Hexagonal)
- **AWS Lambda** for serverless compute
- **DynamoDB** for data persistence
- **Cognito** for authentication
- **Stripe** for payments
- **OpenAI** for resume optimization

## Tech Stack

- Go 1.21
- AWS SDK v2
- Stripe Go SDK
- Chi Router
- Terraform
- GitHub Actions

## Project Structure

```
├── cmd/api/              # Application entry point
├── internal/
│   ├── core/            # Domain entities and ports
│   ├── application/     # Use case implementations
│   └── adapters/        # Inbound/Outbound adapters
├── terraform/           # Infrastructure as Code
├── scripts/             # Utility scripts
└── .github/workflows/   # CI/CD pipelines
```

## Getting Started

### Prerequisites

- Go 1.21+
- AWS CLI configured
- Terraform 1.6+
- Stripe CLI (for webhook testing)

### Installation

```bash
# Clone the repository
git clone https://github.com/reangeline/backend_applywise.git
cd backend_applywise

# Run setup script
chmod +x scripts/setup.sh
./scripts/setup.sh

# Install dependencies
make deps
```

### Configuration

1. Copy `.env.example` to `.env`
2. Fill in your AWS, Stripe, and OpenAI credentials
3. Create Stripe products (run once):
   ```bash
   chmod +x scripts/create-stripe-products.sh
   ./scripts/create-stripe-products.sh
   ```

### Running Locally

```bash
make run-local
```

Server will start on `http://localhost:8080`

### Running Tests

```bash
make test
```

### Deploying

```bash
# Build
make build

# Deploy to AWS
make deploy
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/signup` - Create account
- `POST /api/v1/auth/signin` - Login
- `POST /api/v1/auth/refresh` - Refresh token

### Resumes
- `POST /api/v1/resumes` - Upload resume
- `GET /api/v1/resumes` - List resumes
- `POST /api/v1/resumes/optimize` - Optimize resume

### Subscriptions
- `GET /api/v1/subscription` - Get subscription
- `POST /api/v1/subscription` - Create subscription
- `DELETE /api/v1/subscription` - Cancel subscription

### Webhooks
- `POST /api/v1/webhooks/stripe` - Stripe webhooks

## Contributing

1. Create a feature branch
2. Make your changes
3. Run tests and linting
4. Submit a pull request

## License

Proprietary - All rights reserved