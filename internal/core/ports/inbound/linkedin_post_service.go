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

// DraftPostRequest pede o rascunho de um post pra um tema específico. TopicIndex identifica
// qual tema (na lista de LinkedInPostIdeas já salva do usuário) recebe o rascunho gerado —
// usado só pra persistir o resultado, não afeta a geração em si.
type DraftPostRequest struct {
	UserID     string
	ResumeID   string
	TopicTitle string
	TopicAngle string
	TopicIndex int
}

// DraftPostResult é o post pronto pra copiar.
type DraftPostResult struct {
	PostText string `json:"post_text"`
}

// LinkedInPostService sugere temas de publicação pro LinkedIn com base no currículo do
// usuário, e rascunha posts prontos pros temas escolhidos — spec 018. Temas e o rascunho
// mais recente de cada um ficam salvos (mais recente por usuário, sem histórico) — persistir
// o rascunho é da spec de cache de conteúdo de IA (evita regenerar ao reabrir/trocar tema).
// Sem crédito, só autenticado (mesma decisão da spec 013).
type LinkedInPostService interface {
	GenerateTopics(ctx context.Context, req GenerateTopicsRequest) (*domain.LinkedInPostIdeas, error)
	GetLatestTopics(ctx context.Context, userID string) (*domain.LinkedInPostIdeas, error)
	DraftPost(ctx context.Context, req DraftPostRequest) (*DraftPostResult, error)
}
