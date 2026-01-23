package inbound

import "context"

type CreatePaymentMethodRequest struct {
	UserID       string
	PaymentToken string
}

type CreateCheckoutSessionRequest struct {
	UserID     string
	Plan       string
	SuccessURL string
	CancelURL  string
}

// PaymentService define os casos de uso de pagamentos
type PaymentService interface {
	CreateCustomer(ctx context.Context, userID, email, name string) (string, error)
	CreatePaymentMethod(ctx context.Context, req CreatePaymentMethodRequest) (string, error)
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (string, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}
