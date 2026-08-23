package outbound

import "context"

type ResumeAnalysis struct {
	Skills         []string               `json:"skills"`
	Experience     []string               `json:"experience"`
	Education      []string               `json:"education"`
	Keywords       []string               `json:"keywords"`
	StructuredData map[string]interface{} `json:"structured_data"`
}

type JobAnalysis struct {
	RequiredSkills []string `json:"required_skills"`
	Keywords       []string `json:"keywords"`
	Experience     string   `json:"experience"`
}

type OptimizationResult struct {
	OptimizedContent    string                 `json:"optimized_content"`
	ParsedData          map[string]interface{} `json:"parsed_data"`
	MatchScore          float64                `json:"match_score"`
	Suggestions         []string               `json:"suggestions"`
	MissingRequirements []string               `json:"missing_requirements"`
}

// SalaryEstimate representa a estimativa salarial para um cargo/empresa.
// Quando não há dados confiáveis, Found é false e todos os valores são zero/vazio.
type SalaryEstimate struct {
	Found      bool    `json:"found"`
	Currency   string  `json:"currency,omitempty"`
	MinSalary  float64 `json:"min_salary,omitempty"`
	MaxSalary  float64 `json:"max_salary,omitempty"`
	Midpoint   float64 `json:"midpoint,omitempty"`
	Period     string  `json:"period,omitempty"` // "monthly" or "yearly"
	Location   string  `json:"location,omitempty"`
	Seniority  string  `json:"seniority,omitempty"`
	Notes      string  `json:"notes,omitempty"`
	Disclaimer string  `json:"disclaimer,omitempty"`
}

// LinkedInExperience represents one work experience item in the LinkedIn profile.
type LinkedInExperience struct {
	Role        string   `json:"role"`
	Company     string   `json:"company"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date,omitempty"`
	IsCurrent   bool     `json:"is_current"`
	Description []string `json:"description"`
}

// LinkedInLanguage represents a language entry in the LinkedIn profile.
type LinkedInLanguage struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// LinkedInOptimizationResult is the structured output of a LinkedIn profile optimization.
type LinkedInOptimizationResult struct {
	Headline             string               `json:"headline"`
	About                string               `json:"about"`
	Experiences          []LinkedInExperience `json:"experiences"`
	Skills               []string             `json:"skills"`
	Languages            []LinkedInLanguage   `json:"languages"`
	Suggestions          []string             `json:"suggestions"`
	ProfileStrengthScore float64              `json:"profile_strength_score"`
}

// PDFResumeData represents the structured resume data extracted from a PDF.
type PDFResumeData struct {
	Personal        map[string]interface{}   `json:"personal"`
	Experiences     []map[string]interface{} `json:"experiences"`
	Education       []map[string]interface{} `json:"education"`
	Projects        []map[string]interface{} `json:"projects"`
	Languages       []map[string]interface{} `json:"languages"`
	ATSScore        float64                  `json:"ats_score"`
	ATSImprovements []string                 `json:"ats_improvements"`
}

// CoachJobInput holds all context needed to generate stage-specific coaching content.
type CoachJobInput struct {
	Stage            string
	JobTitle         string
	CompanyName      string
	Location         string
	AtsScore         int
	JobDescription   string
	JobURL           string
	MatchedKeywords  []string
	MissingKeywords  []string
	DaysSinceApplied int
	Tone             string // "default" | "formal" | "shorter"
}

// CoachResult holds the AI-generated coaching content.
type CoachResult struct {
	Content string
	Type    string // "followup" | "interview_prep" | "offer_insights" | "feedback_request"
}

// AIService define integração com serviço de IA
type AIService interface {
	ParseResume(ctx context.Context, content string) (*ResumeAnalysis, error)
	ParseJobDescription(ctx context.Context, content string) (*JobAnalysis, error)
	OptimizeResume(ctx context.Context, resume *ResumeAnalysis, job *JobAnalysis, originalResume string) (*OptimizationResult, error)
	EstimateSalary(ctx context.Context, targetRole, targetCompany string) (*SalaryEstimate, error)
	OptimizeForLinkedIn(ctx context.Context, resume *ResumeAnalysis) (*LinkedInOptimizationResult, error)
	ParseResumeFromText(ctx context.Context, text string) (*PDFResumeData, error)
	GenerateCoachContent(ctx context.Context, input *CoachJobInput) (*CoachResult, error)
}
