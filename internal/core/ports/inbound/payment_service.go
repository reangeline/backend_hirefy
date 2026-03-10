package inbound

import "context"

type CreatePaymentMethodRequest struct {
	UserID       string
	PaymentToken string
}

type CreateCheckoutSessionRequest struct {
	UserID     string // ← Adicionar
	Email      string // ← Adicionar
	Plan       string
	SuccessURL string // ← Pode remover ou manter para flexibilidade futura
	CancelURL  string // ← Pode remover ou manter para flexibilidade futura
}

// PaymentService define os casos de uso de pagamentos
type PaymentService interface {
	CreateCustomer(ctx context.Context, userID, email, name string) (string, error)
	CreatePaymentMethod(ctx context.Context, req CreatePaymentMethodRequest) (string, error)
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (string, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}
