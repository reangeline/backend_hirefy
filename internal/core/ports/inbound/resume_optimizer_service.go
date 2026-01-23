package inbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

type UploadResumeRequest struct {
	UserID  string
	Content string
	FileKey string
}

type OptimizeResumeRequest struct {
	UserID         string
	ResumeID       string
	JobDescription string
}

// ResumeOptimizerService define os casos de uso de otimização de currículo
type ResumeOptimizerService interface {
	UploadResume(ctx context.Context, req UploadResumeRequest) (*domain.Resume, error)
	GetResume(ctx context.Context, userID, resumeID string) (*domain.Resume, error)
	ListResumes(ctx context.Context, userID string) ([]*domain.Resume, error)

	OptimizeResume(ctx context.Context, req OptimizeResumeRequest) (*domain.OptimizedResume, error)
	GetOptimizedResume(ctx context.Context, userID, optimizedResumeID string) (*domain.OptimizedResume, error)
	ListOptimizedResumes(ctx context.Context, userID string) ([]*domain.OptimizedResume, error)
}
