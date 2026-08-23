package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// PipelineRepository persists tracked job applications.
type PipelineRepository interface {
	Create(ctx context.Context, job *domain.PipelineJob) error
	Get(ctx context.Context, userID, jobID string) (*domain.PipelineJob, error)
	List(ctx context.Context, userID string) ([]domain.PipelineJob, error)
	Update(ctx context.Context, job *domain.PipelineJob) error
	Delete(ctx context.Context, userID, jobID string) error
	AppendTimelineEvent(ctx context.Context, userID, jobID string, event domain.TimelineEvent) error
}
