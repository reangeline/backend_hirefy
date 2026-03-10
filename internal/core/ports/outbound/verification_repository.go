package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

type VerificationRepository interface {
	Create(ctx context.Context, code *domain.VerificationCode) error
	GetByEmail(ctx context.Context, email string) (*domain.VerificationCode, error)
	Delete(ctx context.Context, email string) error
}
