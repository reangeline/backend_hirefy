package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// OptimizationJobRepository persists async optimization job state.
type OptimizationJobRepository interface {
	Create(ctx context.Context, job *domain.OptimizationJob) error
	UpdateStatus(ctx context.Context, userID, jobID string, status domain.OptimizationJobStatus, errMsg, optimizedResumeID string) error
	Get(ctx context.Context, userID, jobID string) (*domain.OptimizationJob, error)
}
