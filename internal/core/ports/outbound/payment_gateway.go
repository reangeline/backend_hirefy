package outbound

import "context"

type PaymentGateway interface {
	CreateCustomer(ctx context.Context, email, name string) (string, error)
	CreateSubscription(ctx context.Context, customerID, priceID string) (string, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	VerifyWebhookSignature(payload []byte, signature, secret string) error
	CreateCheckoutSession(ctx context.Context, userID, email, priceID string) (string, error)
}
