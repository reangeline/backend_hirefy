package config

type Config struct {
	DynamoDBTable             string
	CognitoUserPoolID         string
	CognitoClientID           string
	ResumesBucketName         string
	StripeSecretKey           string
	StripeWebhookSecret       string
	StripePricePremiumMonthly string
	WebAppBaseURL             string
	OpenAIKey                 string
	OptimizationQueueURL      string
	FirebaseCredentials       string
	FirebaseProjectID         string
}
