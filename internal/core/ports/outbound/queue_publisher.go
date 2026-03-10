package outbound

import "context"

const (
	JobTypeOptimize = "optimize"
	JobTypeLinkedIn = "linkedin"
)

// OptimizationJobMessage is the payload sent to the queue for async processing.
type OptimizationJobMessage struct {
	JobID          string `json:"job_id"`
	UserID         string `json:"user_id"`
	ResumeID       string `json:"resume_id"`
	JobDescription string `json:"job_description,omitempty"`
	TargetCompany  string `json:"target_company,omitempty"`
	TargetRole     string `json:"target_role,omitempty"`
	// JobType discriminates between processing flows.
	// Empty or "optimize" = regular resume optimization.
	// "linkedin" = LinkedIn profile optimization.
	JobType string `json:"job_type,omitempty"`
}

// QueuePublisher publishes optimization jobs to a message queue (e.g., SQS).
type QueuePublisher interface {
	PublishOptimizationJob(ctx context.Context, msg OptimizationJobMessage) error
}
