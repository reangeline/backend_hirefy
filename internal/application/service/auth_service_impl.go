package service

import (
	"context"
	"fmt"
	"time"

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

	var accessToken, refreshToken, idToken string
	var expiresIn int

	// Aguardar processamento do Cognito
	time.Sleep(1 * time.Second)

	err = nil
	accessToken, refreshToken, idToken, expiresIn, err = s.authProvider.SignIn(ctx, req.Email, req.Password)

	if err != nil {
		return &inbound.AuthResponse{
			Message: "Account created! Please sign in and verify your email.",
		}, nil
	}

	return &inbound.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    expiresIn,
		Message:      "Account created! Please verify your email to unlock all features.",
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

func (s *authServiceImpl) ConfirmSignUp(ctx context.Context, req inbound.ConfirmSignUpRequest) error {
	// Validações
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Code == "" {
		return fmt.Errorf("confirmation code is required")
	}

	// Confirmar no Cognito
	if err := s.authProvider.ConfirmSignUp(ctx, req.Email, req.Code); err != nil {
		return err
	}

	// TODO (Opcional): Atualizar status do usuário no DynamoDB como "verified"
	// user, err := s.userRepo.GetByEmail(ctx, req.Email)
	// if err == nil {
	//     user.EmailVerified = true
	//     s.userRepo.Update(ctx, user)
	// }

	return nil
}

func (s *authServiceImpl) ResendConfirmationCode(ctx context.Context, req inbound.ResendCodeRequest) error {
	// Validação
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	// Reenviar código via Cognito
	if err := s.authProvider.ResendConfirmationCode(ctx, req.Email); err != nil {
		return err
	}

	return nil
}
