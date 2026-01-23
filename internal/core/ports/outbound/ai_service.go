package outbound

import "context"

type ResumeAnalysis struct {
	Skills     []string
	Experience []string
	Education  []string
	Keywords   []string
}

type JobAnalysis struct {
	RequiredSkills []string
	Keywords       []string
	Experience     string
}

type OptimizationResult struct {
	OptimizedContent string   `json:"optimized_content"`
	MatchScore       float64  `json:"match_score"`
	Suggestions      []string `json:"suggestions"`
}

// AIService define integração com serviço de IA
type AIService interface {
	ParseResume(ctx context.Context, content string) (*ResumeAnalysis, error)
	ParseJobDescription(ctx context.Context, content string) (*JobAnalysis, error)
	OptimizeResume(ctx context.Context, resume *ResumeAnalysis, job *JobAnalysis, originalResume string) (*OptimizationResult, error)
}
