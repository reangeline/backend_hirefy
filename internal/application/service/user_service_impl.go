package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type userServiceImpl struct {
	userRepo         outbound.UserRepository
	resumeRepo       outbound.ResumeRepository
	objectStorage    outbound.ObjectStorage
	authProvider     outbound.AuthProvider
	subscriptionRepo outbound.SubscriptionRepository
	verificationRepo outbound.VerificationRepository
}

// NewUserService cria nova instância do serviço de usuários
func NewUserService(
	userRepo outbound.UserRepository,
	resumeRepo outbound.ResumeRepository,
	objectStorage outbound.ObjectStorage,
	authProvider outbound.AuthProvider,
	subscriptionRepo outbound.SubscriptionRepository,
	verificationRepo outbound.VerificationRepository,
) inbound.UserService {
	return &userServiceImpl{
		userRepo:         userRepo,
		resumeRepo:       resumeRepo,
		objectStorage:    objectStorage,
		authProvider:     authProvider,
		subscriptionRepo: subscriptionRepo,
		verificationRepo: verificationRepo,
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

func (s *userServiceImpl) UpdateFCMToken(ctx context.Context, userID, token string) error {
	return s.userRepo.UpdateFCMToken(ctx, userID, token)
}

func (s *userServiceImpl) DeleteUser(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	objectKeys, err := s.collectResumeObjectKeys(ctx, userID)
	if err != nil {
		return err
	}

	// Purge every DynamoDB item under USER#<id>: user record, subscription,
	// resumes, optimized resumes, optimization jobs, pipeline jobs, credits.
	if err := s.userRepo.PurgeAllUserItems(ctx, userID); err != nil {
		return fmt.Errorf("failed to purge user data: %w", err)
	}

	// Clean up the verification code (lives under a different PK).
	if err := s.verificationRepo.Delete(ctx, user.Email); err != nil {
		fmt.Printf("⚠️ Warning: failed to delete verification code for user %s: %v\n", userID, err)
	}

	if len(objectKeys) > 0 && s.objectStorage != nil {
		if err := s.objectStorage.DeleteObjects(ctx, objectKeys); err != nil {
			return fmt.Errorf("failed to delete resume objects: %w", err)
		}
	}

	// Delete the Cognito account after the data purge so a transient auth error
	// cannot leave old cards behind in DynamoDB.
	if err := s.authProvider.DeleteUser(ctx, user.Email); err != nil {
		return fmt.Errorf("failed to delete auth account after data purge: %w", err)
	}

	return nil
}

func (s *userServiceImpl) collectResumeObjectKeys(ctx context.Context, userID string) ([]string, error) {
	if s.resumeRepo == nil {
		return nil, nil
	}

	resumes, err := s.resumeRepo.ListResumesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resumes for deletion: %w", err)
	}

	optimizedResumes, err := s.resumeRepo.ListOptimizedResumesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list optimized resumes for deletion: %w", err)
	}

	seen := make(map[string]struct{})
	objectKeys := make([]string, 0, len(resumes)+len(optimizedResumes))

	for _, resume := range resumes {
		if resume == nil || resume.OriginalS3Key == "" {
			continue
		}
		if _, exists := seen[resume.OriginalS3Key]; exists {
			continue
		}
		seen[resume.OriginalS3Key] = struct{}{}
		objectKeys = append(objectKeys, resume.OriginalS3Key)
	}

	for _, optimized := range optimizedResumes {
		if optimized == nil || optimized.OptimizedS3Key == "" {
			continue
		}
		if _, exists := seen[optimized.OptimizedS3Key]; exists {
			continue
		}
		seen[optimized.OptimizedS3Key] = struct{}{}
		objectKeys = append(objectKeys, optimized.OptimizedS3Key)
	}

	return objectKeys, nil
}
