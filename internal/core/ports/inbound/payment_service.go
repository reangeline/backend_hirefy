package inbound

import "context"

type CreatePaymentMethodRequest struct {
	UserID       string
	PaymentToken string
}

// PaymentService define os casos de uso de pagamentos. CreateCheckoutSession não vive
// aqui — o caminho real de checkout é SubscriptionService.CreateCheckoutSession (usado
// pelo SubscriptionHandler); esta interface só cuida do webhook e de operações avulsas de
// customer/payment method (ver .spec/007-stripe-billing na web-app, achado de código morto).
type PaymentService interface {
	CreateCustomer(ctx context.Context, userID, email, name string) (string, error)
	CreatePaymentMethod(ctx context.Context, req CreatePaymentMethodRequest) (string, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}
