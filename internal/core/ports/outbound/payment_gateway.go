package outbound

import "context"

// PaymentGateway define integração com gateway de pagamento (Stripe)
type PaymentGateway interface {
	CreateCustomer(ctx context.Context, email, name string, metadata map[string]string) (string, error)
	CreateSubscription(ctx context.Context, customerID, priceID, paymentMethodID string) (string, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error)
	VerifyWebhookSignature(payload []byte, signature, secret string) error
}
