package domain

import "time"

type Resume struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Type            string                 `json:"type"`
	OriginalContent string                 `json:"original_content"`
	OriginalS3Key   string                 `json:"original_s3_key,omitempty"`
	ParsedData      map[string]interface{} `json:"parsed_data"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type JobDescription struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Content     string                 `json:"content"`
	ParsedData  map[string]interface{} `json:"parsed_data"`
	CompanyName string                 `json:"company_name,omitempty"`
	JobTitle    string                 `json:"job_title,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type OptimizedResume struct {
	ID                  string                 `json:"id"`
	UserID              string                 `json:"user_id"`
	ResumeID            string                 `json:"resume_id"`
	SourceResumeID      string                 `json:"source_resume_id"`
	JobDescriptionID    string                 `json:"job_description_id"`
	OptimizedContent    string                 `json:"optimized_content"`
	OptimizedS3Key      string                 `json:"optimized_s3_key,omitempty"`
	ParsedData          map[string]interface{} `json:"parsed_data"`
	MatchScore          float64                `json:"match_score"`
	Suggestions         []string               `json:"suggestions"`
	MissingRequirements []string               `json:"missing_requirements"`
	SalaryEstimate      *SalaryEstimate        `json:"salary_estimate,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

// SalaryEstimate representa a faixa salarial estimada para o cargo/empresa alvo.
type SalaryEstimate struct {
	Found      bool    `json:"found"`
	Currency   string  `json:"currency,omitempty"`
	MinSalary  float64 `json:"min_salary,omitempty"`
	MaxSalary  float64 `json:"max_salary,omitempty"`
	Midpoint   float64 `json:"midpoint,omitempty"`
	Period     string  `json:"period,omitempty"`
	Location   string  `json:"location,omitempty"`
	Seniority  string  `json:"seniority,omitempty"`
	Notes      string  `json:"notes,omitempty"`
	Disclaimer string  `json:"disclaimer,omitempty"`
}
