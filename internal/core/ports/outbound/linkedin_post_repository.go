package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// LinkedInPostIdeasRepository persiste o conjunto de temas de post mais recente por
// usuário — sem histórico, um registro é sobrescrito pelo próximo (spec 018).
type LinkedInPostIdeasRepository interface {
	Upsert(ctx context.Context, ideas *domain.LinkedInPostIdeas) error
	GetByUserID(ctx context.Context, userID string) (*domain.LinkedInPostIdeas, error)
}
