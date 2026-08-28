package inbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

type CreateSubscriptionRequest struct {
	UserID        string
	Plan          domain.SubscriptionPlan
	PaymentMethod string // Stripe payment method ID
}

// CreateCheckoutRequest — sem PriceID de propósito: o cliente não escolhe o preço, o
// backend sempre usa o Price ID do Premium configurado no servidor (ver
// .spec/007-stripe-billing na web-app). Só existe 1 plano pago hoje.
type CreateCheckoutRequest struct {
	UserID string
	Email  string
}

type SubscriptionService interface {
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*domain.Subscription, error)
	GetSubscriptionByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
	CancelSubscription(ctx context.Context, userID string) error
	UpdateSubscription(ctx context.Context, userID string, newPlan domain.SubscriptionPlan) (*domain.Subscription, error)
	CheckSubscriptionStatus(ctx context.Context, userID string) (bool, error)
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (string, error)
	GetCreditHistory(ctx context.Context, userID string, limit int) ([]*domain.CreditTransaction, error)
}
