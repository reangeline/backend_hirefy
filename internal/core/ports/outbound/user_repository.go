package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// UserRepository define como persistir usuários
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, userID string) (*domain.User, error)
	GetByCognitoID(ctx context.Context, cognitoID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateFCMToken(ctx context.Context, userID, token string) error
	Delete(ctx context.Context, userID string) error
}
