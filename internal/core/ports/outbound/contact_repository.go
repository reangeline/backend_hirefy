package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// ContactRepository persists contacts linked to pipeline jobs.
type ContactRepository interface {
	List(ctx context.Context, userID, jobID string) ([]domain.PipelineContact, error)
	Add(ctx context.Context, contact *domain.PipelineContact) error
	Delete(ctx context.Context, userID, jobID, contactID string) error
}
