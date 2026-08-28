package outbound

import "context"

type PaymentGateway interface {
	CreateCustomer(ctx context.Context, email, name string) (string, error)
	CreateSubscription(ctx context.Context, customerID, priceID string) (string, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	VerifyWebhookSignature(payload []byte, signature, secret string) error
	CreateCheckoutSession(ctx context.Context, userID, email, priceID string) (string, error)
	// GetSubscriptionPriceID busca o Price ID real de uma subscription no Stripe — usado
	// pelo webhook pra confirmar qual plano foi comprado, em vez de confiar cegamente no
	// payload do evento (ver .spec/007-stripe-billing na web-app).
	GetSubscriptionPriceID(ctx context.Context, subscriptionID string) (string, error)
}
