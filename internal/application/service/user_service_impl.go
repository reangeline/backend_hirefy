package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type userServiceImpl struct {
	userRepo outbound.UserRepository
}

// NewUserService cria nova instância do serviço de usuários
func NewUserService(userRepo outbound.UserRepository) inbound.UserService {
	return &userServiceImpl{
		userRepo: userRepo,
	}
}

func (s *userServiceImpl) CreateUser(ctx context.Context, email, name, cognitoID string) (*domain.User, error) {
	// Verifica se usuário já existe
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Cria novo usuário
	user := domain.NewUser(email, name, cognitoID)
	user.ID = uuid.New().String()

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceImpl) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *userServiceImpl) GetUserByCognitoID(ctx context.Context, cognitoID string) (*domain.User, error) {
	return s.userRepo.GetByCognitoID(ctx, cognitoID)
}

func (s *userServiceImpl) UpdateUser(ctx context.Context, userID string, name string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Name = name
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}
