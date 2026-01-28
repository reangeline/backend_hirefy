package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	"github.com/aws/aws-sdk-go-v2/service/sesv2" // ✅ ADICIONAR
	httpAdapter "github.com/reangeline/backend_applywise/internal/adapters/inbound/http"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/ai/openai"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/auth/cognito"
	emailadapter "github.com/reangeline/backend_applywise/internal/adapters/outbound/email" // ✅ ADICIONAR
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/payment/stripe"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/persistence/dynamodb"
	appservice "github.com/reangeline/backend_applywise/internal/application/service"
	appconfig "github.com/reangeline/backend_applywise/pkg/config"
)

var chiLambda *chiadapter.ChiLambda

func init() {
	// Carrega configurações
	cfg := loadConfig()

	// Inicializa AWS SDK
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("failed to load AWS config:", err)
	}

	// Inicializa adapters (outbound)
	dynamoClient := dynamodb.NewClient(awsCfg, cfg.DynamoDBTable)
	sesClient := sesv2.NewFromConfig(awsCfg)

	userRepo := dynamodb.NewUserRepository(dynamoClient)
	subscriptionRepo := dynamodb.NewSubscriptionRepository(dynamoClient)
	resumeRepo := dynamodb.NewResumeRepository(dynamoClient)
	verificationRepo := dynamodb.NewVerificationRepository(dynamoClient) // ✅ ADICIONAR

	emailService := emailadapter.NewEmailService(emailadapter.EmailConfig{
		Client:    sesClient,
		FromEmail: os.Getenv("SES_FROM_EMAIL"),
		FromName:  "ApplyWise",
		IsDev:     os.Getenv("ENVIRONMENT") == "dev",
	})

	cognitoClient := cognito.NewAuthProvider(awsCfg, cfg.CognitoUserPoolID, cfg.CognitoClientID)
	stripeClient := stripe.NewPaymentGateway(cfg.StripeSecretKey)
	aiClient := openai.NewAIService(cfg.OpenAIKey)

	// Inicializa services (application layer)
	userService := appservice.NewUserService(userRepo)

	authService := appservice.NewAuthService(
		cognitoClient,
		userRepo,
		subscriptionRepo,
		verificationRepo,
		emailService,
	)

	subscriptionService := appservice.NewSubscriptionService(subscriptionRepo, stripeClient, userRepo)
	paymentService := appservice.NewPaymentService(stripeClient, subscriptionRepo, cfg.StripeWebhookSecret)
	resumeService := appservice.NewResumeOptimizerService(resumeRepo, aiClient, subscriptionRepo)

	revenueCatService := appservice.NewRevenueCatService(
		subscriptionRepo,
		userRepo,
	)

	// Inicializa router (inbound adapter)
	router := httpAdapter.NewRouter(
		authService,
		userService,
		subscriptionService,
		resumeService,
		paymentService,
		revenueCatService,
	)

	// Configura Lambda adapter
	chiLambda = chiadapter.New(router)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return chiLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Se está rodando localmente, usa servidor HTTP
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		runLocalServer()
	} else {
		// Se está na Lambda, usa o handler
		lambda.Start(Handler)
	}
}

func runLocalServer() {
	cfg := loadConfig()

	// ... (mesmo código de init mas sem chiLambda)
	awsCfg, _ := config.LoadDefaultConfig(context.Background())
	dynamoClient := dynamodb.NewClient(awsCfg, cfg.DynamoDBTable)

	userRepo := dynamodb.NewUserRepository(dynamoClient)
	subscriptionRepo := dynamodb.NewSubscriptionRepository(dynamoClient)
	resumeRepo := dynamodb.NewResumeRepository(dynamoClient)
	verificationRepo := dynamodb.NewVerificationRepository(dynamoClient)
	cognitoClient := cognito.NewAuthProvider(awsCfg, cfg.CognitoUserPoolID, cfg.CognitoClientID)
	stripeClient := stripe.NewPaymentGateway(cfg.StripeSecretKey)
	aiClient := openai.NewAIService(cfg.OpenAIKey)

	sesClient := sesv2.NewFromConfig(awsCfg)

	emailService := emailadapter.NewEmailService(emailadapter.EmailConfig{
		Client:    sesClient,
		FromEmail: os.Getenv("SES_FROM_EMAIL"),
		FromName:  "ApplyWise",
		IsDev:     os.Getenv("ENVIRONMENT") == "dev",
	})

	userService := appservice.NewUserService(userRepo)
	authService := appservice.NewAuthService(cognitoClient, userRepo, subscriptionRepo, verificationRepo, emailService)

	subscriptionService := appservice.NewSubscriptionService(subscriptionRepo, stripeClient, userRepo)
	paymentService := appservice.NewPaymentService(stripeClient, subscriptionRepo, cfg.StripeWebhookSecret)

	revenueCatService := appservice.NewRevenueCatService(
		subscriptionRepo,
		userRepo,
	)

	resumeService := appservice.NewResumeOptimizerService(resumeRepo, aiClient, subscriptionRepo)

	router := httpAdapter.NewRouter(
		authService,
		userService,
		subscriptionService,
		resumeService,
		paymentService,
		revenueCatService,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() *appconfig.Config {
	return &appconfig.Config{
		DynamoDBTable:       os.Getenv("DYNAMODB_TABLE"),
		CognitoUserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		OpenAIKey:           os.Getenv("OPENAI_API_KEY"),
	}
}
