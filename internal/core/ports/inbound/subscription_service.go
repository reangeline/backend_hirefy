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

// SubscriptionService define os casos de uso de assinaturas
type SubscriptionService interface {
	CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*domain.Subscription, error)
	GetSubscriptionByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
	CancelSubscription(ctx context.Context, userID string) error
	UpdateSubscription(ctx context.Context, userID string, newPlan domain.SubscriptionPlan) (*domain.Subscription, error)
	CheckSubscriptionStatus(ctx context.Context, userID string) (bool, error)
}
