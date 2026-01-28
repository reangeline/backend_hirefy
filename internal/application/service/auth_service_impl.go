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
	authProvider     outbound.AuthProvider
	userRepo         outbound.UserRepository
	subscriptionRepo outbound.SubscriptionRepository
	verificationRepo outbound.VerificationRepository // ✅ ADICIONAR
	emailService     outbound.EmailService           // ✅ ADICIONAR
}

func NewAuthService(
	authProvider outbound.AuthProvider,
	userRepo outbound.UserRepository,
	subscriptionRepo outbound.SubscriptionRepository,
	verificationRepo outbound.VerificationRepository, // ✅ ADICIONAR
	emailService outbound.EmailService,
) inbound.AuthService {
	return &authServiceImpl{
		authProvider:     authProvider,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		verificationRepo: verificationRepo, // ✅ ADICIONAR
		emailService:     emailService,     // ✅ ADICIONAR
	}
}

func (s *authServiceImpl) SignUp(ctx context.Context, req inbound.SignUpRequest) (*inbound.AuthResponse, error) {
	// Valida se usuário já existe
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// 1. Cria usuário no Cognito
	cognitoID, err := s.authProvider.SignUp(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in Cognito: %w", err)
	}

	// 2. Criar usuário no DynamoDB
	user := domain.NewUser(req.Email, req.Name, cognitoID)
	user.ID = cognitoID
	user.EmailVerified = false

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create local user: %w", err)
	}

	// 3. Criar subscription FREE
	subscription := domain.NewSubscription(user.ID, domain.PlanFree)
	subscription.ID = uuid.New().String()

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// 4. ✅ Gerar código de verificação customizado
	verificationCode := domain.NewVerificationCode(req.Email)
	if err := s.verificationRepo.Create(ctx, verificationCode); err != nil {
		fmt.Printf("Warning: failed to create verification code: %v\n", err)
	}

	// 5. ✅ Enviar email com código
	if err := s.emailService.SendVerificationEmail(ctx, req.Email, verificationCode.Code); err != nil {
		fmt.Printf("Warning: failed to send verification email: %v\n", err)
	}

	// 6. Fazer login automático
	time.Sleep(500 * time.Millisecond)

	accessToken, refreshToken, idToken, expiresIn, err := s.authProvider.SignIn(ctx, req.Email, req.Password)
	if err != nil {
		return &inbound.AuthResponse{
			Message: "Account created! Please check your email and sign in.",
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
	if req.Email == "" || req.Code == "" {
		return fmt.Errorf("email and code are required")
	}

	// 1. Buscar código no DynamoDB
	verificationCode, err := s.verificationRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("verification code not found or expired")
	}

	// 2. Verificar se expirou
	if verificationCode.IsExpired() {
		s.verificationRepo.Delete(ctx, req.Email)
		return fmt.Errorf("verification code expired")
	}

	// 3. Validar código
	if verificationCode.Code != req.Code {
		return fmt.Errorf("invalid verification code")
	}

	// 4. Deletar código usado
	s.verificationRepo.Delete(ctx, req.Email)

	// 5. Marcar email como verificado no DynamoDB
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 6. ✅ ADICIONAR: Marcar no Cognito também
	if err := s.authProvider.MarkEmailAsVerified(ctx, req.Email); err != nil {
		// Apenas log, não falhar por isso
		fmt.Printf("⚠️ Warning: failed to update Cognito email_verified: %v\n", err)
	}

	return nil
}

func (s *authServiceImpl) ResendConfirmationCode(ctx context.Context, req inbound.ResendCodeRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	// 1. Verificar se user existe
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// 2. Se já está verificado, retornar erro amigável
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// 3. ✅ Deletar código antigo (se existir)
	s.verificationRepo.Delete(ctx, req.Email)

	// 4. ✅ Gerar novo código
	verificationCode := domain.NewVerificationCode(req.Email)
	if err := s.verificationRepo.Create(ctx, verificationCode); err != nil {
		return fmt.Errorf("failed to create verification code: %w", err)
	}

	// 5. ✅ Enviar email
	if err := s.emailService.SendVerificationEmail(ctx, req.Email, verificationCode.Code); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
