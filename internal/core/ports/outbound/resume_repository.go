package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// ResumeRepository define como persistir currículos
type ResumeRepository interface {
	CreateResume(ctx context.Context, resume *domain.Resume) error
	GetResume(ctx context.Context, userID, resumeID string) (*domain.Resume, error)
	ListResumesByUserID(ctx context.Context, userID string) ([]*domain.Resume, error)

	UpdateResume(ctx context.Context, resume *domain.Resume) error
	DeleteResume(ctx context.Context, userID, resumeID string) error
	DeleteOptimizedResume(ctx context.Context, userID, optimizedID string) error

	CreateOptimizedResume(ctx context.Context, optimized *domain.OptimizedResume) error
	UpdateOptimizedResume(ctx context.Context, optimized *domain.OptimizedResume) error
	GetOptimizedResume(ctx context.Context, optimizedID string) (*domain.OptimizedResume, error)
	ListOptimizedResumesByUserID(ctx context.Context, userID string) ([]*domain.OptimizedResume, error)
}
