package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// CoachSuggestionRepository persiste o conteúdo de coach mais recente por (usuário, vaga,
// estágio) — sem histórico, um registro é sobrescrito pelo próximo (mesmo padrão de
// LinkedInPostIdeasRepository/LinkedInScanRepository).
type CoachSuggestionRepository interface {
	Upsert(ctx context.Context, suggestion *domain.CoachSuggestion) error
	// Get retorna domain.ErrCoachSuggestionNotFound se nunca foi gerado pra esse (jobID, stage).
	Get(ctx context.Context, userID, jobID string, stage domain.PipelineJobStage) (*domain.CoachSuggestion, error)
}
