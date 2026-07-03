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
}

// CoachJobResponse is the AI-generated coaching output.
type CoachJobResponse struct {
	Content string `json:"content"`
	Stage   string `json:"stage"`
	Type    string `json:"type"` // "followup" | "interview_prep" | "offer_insights" | "feedback_request"
}

// PipelineCoachService generates stage-specific coaching content for a pipeline job.
type PipelineCoachService interface {
	Coach(ctx context.Context, req CoachJobRequest) (*CoachJobResponse, error)
}
