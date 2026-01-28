package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *domain.Subscription) error
	GetByID(ctx context.Context, subscriptionID string) (*domain.Subscription, error)
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error)
	GetByRevenueCatCustomerID(ctx context.Context, customerID string) (*domain.Subscription, error) // ✅ ADICIONAR
	Update(ctx context.Context, subscription *domain.Subscription) error
}
