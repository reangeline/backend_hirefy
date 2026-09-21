package inbound

import "context"

// CoachJobRequest carries all inputs needed for AI-powered job coaching.
type CoachJobRequest struct {
	UserID           string
	JobID            string
	Stage            string
	JobTitle         string
	CompanyName      string
	Location         string
	AtsScore         int
	ResumeVersion    string
	JobDescription   string
	JobURL           string
	MatchedKeywords  []string
	MissingKeywords  []string
	DaysSinceApplied int
	Tone             string // "default" | "formal" | "shorter"
	// ForceRegenerate ignora qualquer sugestão já salva pra esse (job, stage) e gera (e
	// cobra crédito) de novo — usado só pelo botão "Regenerar" explícito no frontend.
	ForceRegenerate bool
}

// CoachJobResponse is the AI-generated coaching output.
type CoachJobResponse struct {
	Content string `json:"content"`
	Stage   string `json:"stage"`
	Type    string `json:"type"` // "followup" | "interview_prep" | "offer_insights" | "feedback_request"
}

// PipelineCoachService generates stage-specific coaching content for a pipeline job.
type PipelineCoachService interface {
	// Coach retorna a sugestão já salva pra esse (job, stage) sem chamar a IA, a menos que
	// ForceRegenerate seja true ou nada tenha sido gerado ainda — nesses casos gera (cobra
	// crédito no plano Free) e salva o resultado antes de retornar.
	Coach(ctx context.Context, req CoachJobRequest) (*CoachJobResponse, error)
	// GetCachedCoach só lê — nunca gera nem cobra crédito. Retorna
	// domain.ErrCoachSuggestionNotFound se nada foi gerado ainda pra esse (job, stage).
	GetCachedCoach(ctx context.Context, userID, jobID, stage string) (*CoachJobResponse, error)
}
