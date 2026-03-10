package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/reangeline/backend_applywise/internal/adapters/outbound/ai/openai"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/notification/fcm"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/persistence/dynamodb"
	"github.com/reangeline/backend_applywise/internal/application/service"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	appconfig "github.com/reangeline/backend_applywise/pkg/config"
)

type optimizationProcessor struct {
	resumeService inbound.ResumeOptimizerService
}

func (p *optimizationProcessor) Handle(ctx context.Context, event events.SQSEvent) error {
	for _, record := range event.Records {
		var msg outbound.OptimizationJobMessage
		if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
			log.Printf("[worker] invalid message: %v", err)
			continue
		}

		var err error
		switch msg.JobType {
		case outbound.JobTypeLinkedIn:
			_, err = p.resumeService.ProcessLinkedInOptimizationJob(ctx, inbound.ProcessLinkedInOptimizationJobRequest{
				JobID:    msg.JobID,
				UserID:   msg.UserID,
				ResumeID: msg.ResumeID,
			})
		default:
			_, err = p.resumeService.ProcessOptimizationJob(ctx, inbound.ProcessOptimizationJobRequest{
				JobID:          msg.JobID,
				UserID:         msg.UserID,
				ResumeID:       msg.ResumeID,
				JobDescription: msg.JobDescription,
				TargetCompany:  msg.TargetCompany,
				TargetRole:     msg.TargetRole,
			})
		}

		if err != nil {
			log.Printf("[worker] failed processing job %s (type=%q): %v", msg.JobID, msg.JobType, err)
			return err
		}
	}

	return nil
}

func main() {
	cfg := loadConfig()

	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("failed to load AWS config:", err)
	}

	dynamoClient := dynamodb.NewClient(awsCfg, cfg.DynamoDBTable)
	resumeRepo := dynamodb.NewResumeRepository(dynamoClient)
	subscriptionRepo := dynamodb.NewSubscriptionRepository(dynamoClient)
	creditTransactionRepo := dynamodb.NewCreditTransactionRepository(dynamoClient)
	jobRepo := dynamodb.NewOptimizationJobRepository(dynamoClient)
	userRepo := dynamodb.NewUserRepository(dynamoClient)

	if cfg.FirebaseCredentials == "" {
		log.Printf("[worker] FIREBASE_CREDENTIALS_FILE is empty; push notifications will be disabled")
	}

	notifier, err := fcm.NewPublisher(context.Background(), cfg.FirebaseCredentials, cfg.FirebaseProjectID, userRepo)
	if err != nil {
		log.Printf("[worker] failed to init FCM publisher: %v", err)
	}
	if notifier == nil {
		log.Printf("[worker] FCM notifier not initialized (check credentials file path/bundle)")
	}

	aiClient := openai.NewAIService(cfg.OpenAIKey)

	resumeService := service.NewResumeOptimizerService(
		resumeRepo,
		aiClient,
		subscriptionRepo,
		creditTransactionRepo,
		nil, // publisher not needed in worker path
		jobRepo,
		notifier,
	)

	processor := &optimizationProcessor{resumeService: resumeService}

	lambda.Start(processor.Handle)
}

func loadConfig() *appconfig.Config {
	return &appconfig.Config{
		DynamoDBTable:        os.Getenv("DYNAMODB_TABLE"),
		CognitoUserPoolID:    os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoClientID:      os.Getenv("COGNITO_CLIENT_ID"),
		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
		OpenAIKey:            os.Getenv("OPENAI_API_KEY"),
		OptimizationQueueURL: os.Getenv("OPTIMIZATION_QUEUE_URL"),
		FirebaseCredentials:  os.Getenv("FIREBASE_CREDENTIALS_FILE"),
		FirebaseProjectID:    os.Getenv("FIREBASE_PROJECT_ID"),
	}
}
