package domain

import "time"

type Resume struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
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
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	ResumeID         string    `json:"resume_id"`
	JobDescriptionID string    `json:"job_description_id"`
	OptimizedContent string    `json:"optimized_content"`
	OptimizedS3Key   string    `json:"optimized_s3_key,omitempty"`
	MatchScore       float64   `json:"match_score"`
	Suggestions      []string  `json:"suggestions"`
	CreatedAt        time.Time `json:"created_at"`
}
