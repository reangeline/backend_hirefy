package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

type CreditTransactionRepository interface {
	Create(ctx context.Context, transaction *domain.CreditTransaction) error
	ListByUserID(ctx context.Context, userID string, limit int) ([]*domain.CreditTransaction, error)
}
