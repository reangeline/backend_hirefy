package outbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// InterviewRepository persiste as perguntas de prática de entrevista ligadas a vagas do
// pipeline. Mesmo padrão de chave de ContactRepository.
type InterviewRepository interface {
	List(ctx context.Context, userID, jobID string) ([]domain.InterviewQuestion, error)
	Get(ctx context.Context, userID, jobID, questionID string) (*domain.InterviewQuestion, error)
	Create(ctx context.Context, question *domain.InterviewQuestion) error
	Update(ctx context.Context, question *domain.InterviewQuestion) error
}
