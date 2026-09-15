package inbound

import (
	"context"

	"github.com/reangeline/backend_applywise/internal/core/domain"
)

// GenerateTopicsRequest pede uma lista de temas de post sugeridos a partir de um currículo
// salvo (spec 018).
type GenerateTopicsRequest struct {
	UserID     string
	ResumeID   string
	TargetRole string // opcional
}

// DraftPostRequest pede o rascunho de um post pra um tema específico.
type DraftPostRequest struct {
	UserID     string
	ResumeID   string
	TopicTitle string
	TopicAngle string
}

// DraftPostResult é o post pronto pra copiar.
type DraftPostResult struct {
	PostText string `json:"post_text"`
}

// LinkedInPostService sugere temas de publicação pro LinkedIn com base no currículo do
// usuário, e rascunha posts prontos pros temas escolhidos — spec 018. Só a lista de temas é
// persistida (mais recente por usuário, sem histórico); rascunhos de post não são salvos.
// Sem crédito, só autenticado (mesma decisão da spec 013).
type LinkedInPostService interface {
	GenerateTopics(ctx context.Context, req GenerateTopicsRequest) (*domain.LinkedInPostIdeas, error)
	GetLatestTopics(ctx context.Context, userID string) (*domain.LinkedInPostIdeas, error)
	DraftPost(ctx context.Context, req DraftPostRequest) (*DraftPostResult, error)
}
