package inbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// UserService define os casos de uso relacionados a usuários
type UserService interface {
	CreateUser(ctx context.Context, email, name, cognitoID string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetUserByCognitoID(ctx context.Context, cognitoID string) (*domain.User, error)
	UpdateUser(ctx context.Context, userID string, name string) (*domain.User, error)
	DeleteUser(ctx context.Context, userID string) error
}
