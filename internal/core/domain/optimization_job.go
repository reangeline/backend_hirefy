package domain

import "time"

// OptimizationJob represents an async resume optimization job.
type OptimizationJob struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"user_id"`
	ResumeID          string                 `json:"resume_id"`
	JobDescription    string                 `json:"job_description"`
	Status            OptimizationJobStatus  `json:"status"`
	Error             string                 `json:"error,omitempty"`
	OptimizedResumeID string                 `json:"optimized_resume_id,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// OptimizationJobStatus enumerates possible job states.
type OptimizationJobStatus string

const (
	JobStatusQueued     OptimizationJobStatus = "queued"
	JobStatusProcessing OptimizationJobStatus = "processing"
	JobStatusCompleted  OptimizationJobStatus = "completed"
	JobStatusFailed     OptimizationJobStatus = "failed"
)
