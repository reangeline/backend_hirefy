package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/adapters/outbound/auth/social"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type authServiceImpl struct {
	authProvider     outbound.AuthProvider
	userRepo         outbound.UserRepository
	subscriptionRepo outbound.SubscriptionRepository
	verificationRepo outbound.VerificationRepository
	emailService     outbound.EmailService
	resumeRepo       outbound.ResumeRepository
}

func NewAuthService(
	authProvider outbound.AuthProvider,
	userRepo outbound.UserRepository,
	subscriptionRepo outbound.SubscriptionRepository,
	verificationRepo outbound.VerificationRepository,
	emailService outbound.EmailService,
	resumeRepo outbound.ResumeRepository,
) inbound.AuthService {
	return &authServiceImpl{
		authProvider:     authProvider,
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		verificationRepo: verificationRepo,
		emailService:     emailService,
		resumeRepo:       resumeRepo,
	}
}

func (s *authServiceImpl) SignUp(ctx context.Context, req inbound.SignUpRequest) (*inbound.AuthResponse, error) {
	// Valida se usuário já existe
	existingUser, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existingUser != nil {
		// A DynamoDB record exists. Verify whether the Cognito user also exists.
		// If it does not, the record is orphaned (e.g. Cognito was deleted but the
		// DynamoDB cleanup failed). Clean it up and allow the new signup to proceed.
		cognitoExists, err := s.authProvider.UserExistsInCognito(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to verify account state: %w", err)
		}
		if cognitoExists {
			return nil, domain.ErrUserAlreadyExists
		}
		// Orphaned record – purge ALL items (user, subscription, pipelines, resumes,
		// contacts, credit transactions, etc.) before creating the new account.
		fmt.Printf("⚠️ Cleaning up orphaned DynamoDB records for %s (userID=%s)\n", req.Email, existingUser.ID)
		_ = s.verificationRepo.Delete(ctx, existingUser.Email)
		if cleanErr := s.userRepo.PurgeAllUserItems(ctx, existingUser.ID); cleanErr != nil {
			fmt.Printf("⚠️ Failed to purge orphaned records for %s: %v\n", req.Email, cleanErr)
		}
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

	// Salvar consentimento dos termos
	if req.TermsAcceptedAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.TermsAcceptedAt)
		if err == nil {
			user.TermsAcceptedAt = &parsedTime
		}
	}
	user.TermsVersion = req.TermsVersion

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
		fmt.Printf("❌ Failed to save verification code for %s: %v\n", req.Email, err)
		return nil, fmt.Errorf("failed to create verification code: %w", err)
	}

	// 5. ✅ Enviar email com código
	if err := s.emailService.SendVerificationEmail(ctx, req.Email, verificationCode.Code); err != nil {
		fmt.Printf("❌ Failed to send verification email to %s: %v\n", req.Email, err)
		// código já salvo no DynamoDB; usuário pode usar resend depois
	}

	// 6. Criar currículo manual se vier no request
	if len(req.ParsedResume) > 0 {
		resume := &domain.Resume{
			ID:         uuid.New().String(),
			UserID:     cognitoID,
			Type:       "manual",
			ParsedData: req.ParsedResume,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.resumeRepo.CreateResume(ctx, resume); err != nil {
			fmt.Printf("⚠️ Warning: failed to create resume during signup for %s: %v\n", req.Email, err)
		}
	}

	// 7. Fazer login automático
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

func (s *authServiceImpl) ForgotPassword(ctx context.Context, req inbound.ForgotPasswordRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	return s.authProvider.ForgotPassword(ctx, req.Email)
}

func (s *authServiceImpl) ConfirmForgotPassword(ctx context.Context, req inbound.ConfirmForgotPasswordRequest) error {
	if req.Email == "" || req.Code == "" || req.NewPassword == "" {
		return fmt.Errorf("email, code and new_password are required")
	}
	return s.authProvider.ConfirmForgotPassword(ctx, req.Email, req.Code, req.NewPassword)
}

func (s *authServiceImpl) SocialSignIn(ctx context.Context, req inbound.SocialSignInRequest) (*inbound.AuthResponse, error) {
	if req.Provider != "apple" && req.Provider != "google" {
		return nil, fmt.Errorf("unsupported provider %q: must be \"apple\" or \"google\"", req.Provider)
	}
	if req.IDToken == "" {
		return nil, fmt.Errorf("id_token is required")
	}

	// 1. Validate the provider token and extract claims.
	var claims *social.TokenClaims
	var err error
	switch req.Provider {
	case "apple":
		claims, err = social.ValidateAppleToken(ctx, req.IDToken)
	case "google":
		claims, err = social.ValidateGoogleToken(ctx, req.IDToken)
	}
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// 2. Require an email address.
	// Apple omits the email claim after the first authentication. The mobile app
	// must cache the email and pass provider=apple with a fresh token when this
	// error is returned.
	if claims.Email == "" {
		return nil, fmt.Errorf("email not available in token; please sign in with %s again to grant email access", req.Provider)
	}

	// 3. Resolve display name: request body takes priority over the token claim.
	name := req.Name
	if name == "" {
		name = claims.Name
	}
	if name == "" {
		name = claims.Email // last-resort fallback
	}

	// 4. Run the Cognito side first to get the authoritative cognitoID.
	//    This must happen before any DynamoDB lookup so we always have the
	//    definitive identity, avoiding stale GSI reads after account deletion.
	cognitoID, accessToken, refreshToken, idToken, expiresIn, err :=
		s.authProvider.SocialSignIn(ctx, claims.Email, name)
	if err != nil {
		return nil, fmt.Errorf("social authentication failed: %w", err)
	}

	// 5. Look up the user by cognitoID (authoritative — not by email via GSI).
	//    After an account deletion + immediate re-signup, GetByEmail via GSI1
	//    can return stale data (eventually consistent), causing the old user to
	//    be found and the "return tokens" path to be taken, leaving old pipeline
	//    cards visible. Looking up by cognitoID after provisioning is safer:
	//    a brand-new Cognito user will never have a DynamoDB record yet.
	existingUser, _ := s.userRepo.GetByCognitoID(ctx, cognitoID)

	// 6. Provision DynamoDB records when no record exists for this cognitoID.
	if existingUser == nil {
		// Before creating, purge any stale records indexed under this email
		// (e.g. a previous account whose DynamoDB cleanup did not complete).
		if stale, _ := s.userRepo.GetByEmail(ctx, claims.Email); stale != nil && stale.CognitoID != cognitoID {
			fmt.Printf("⚠️ SocialSignIn: purging stale records for %s (staleID=%s)\n", claims.Email, stale.ID)
			if purgeErr := s.userRepo.PurgeAllUserItems(ctx, stale.ID); purgeErr != nil {
				fmt.Printf("⚠️ SocialSignIn: failed to purge stale records for %s: %v\n", claims.Email, purgeErr)
			}
		}

		user := domain.NewUser(claims.Email, name, cognitoID)
		user.ID = cognitoID
		user.EmailVerified = true

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user record: %w", err)
		}

		subscription := domain.NewSubscription(user.ID, domain.PlanFree)
		subscription.ID = uuid.New().String()

		if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	return &inbound.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    expiresIn,
	}, nil
}
