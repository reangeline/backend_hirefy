package config

type Config struct {
	DynamoDBTable       string
	CognitoUserPoolID   string
	CognitoClientID     string
	StripeSecretKey     string
	StripeWebhookSecret string
	OpenAIKey           string
}
