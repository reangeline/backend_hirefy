package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type authServiceImpl struct {
	authProvider outbound.AuthProvider
	userRepo     outbound.UserRepository
}

func NewAuthService(
	authProvider outbound.AuthProvider,
	userRepo outbound.UserRepository,
) inbound.AuthService {
	return &authServiceImpl{
		authProvider: authProvider,
		userRepo:     userRepo,
	}
}

func (s *authServiceImpl) SignUp(ctx context.Context, req inbound.SignUpRequest) (*inbound.AuthResponse, error) {
	// Valida se usuário já existe localmente
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Cria usuário no Cognito e pega o cognitoID
	cognitoID, err := s.authProvider.SignUp(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in Cognito: %w", err)
	}

	// Cria usuário local no DynamoDB
	user := domain.NewUser(req.Email, req.Name, cognitoID)
	user.ID = uuid.New().String()

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create local user: %w", err)
	}

	freeSubscription := domain.NewSubscription(user.ID, domain.PlanFree)
	freeSubscription.ID = uuid.New().String()

	// Faz login automaticamente após signup
	accessToken, refreshToken, idToken, expiresIn, err := s.authProvider.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in after signup: %w", err)
	}

	return &inbound.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *authServiceImpl) SignIn(ctx context.Context, req inbound.SignInRequest) (*inbound.AuthResponse, error) {
	accessToken, refreshToken, idToken, expiresIn, err := s.authProvider.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return &inbound.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *authServiceImpl) SignOut(ctx context.Context, accessToken string) error {
	return s.authProvider.SignOut(ctx, accessToken)
}

func (s *authServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*inbound.AuthResponse, error) {
	accessToken, idToken, expiresIn, err := s.authProvider.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return &inbound.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *authServiceImpl) VerifyToken(ctx context.Context, token string) (string, error) {
	cognitoID, err := s.authProvider.VerifyToken(ctx, token)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	return cognitoID, nil
}
