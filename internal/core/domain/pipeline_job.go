package domain

import (
	"strings"
	"time"
)

// PipelineJobStage represents the kanban stage of a job application.
type PipelineJobStage string

const (
	StageWishlist  PipelineJobStage = "wishlist"
	StageApplied   PipelineJobStage = "applied"
	StageInterview PipelineJobStage = "interview"
	StageOffer     PipelineJobStage = "offer"
	StageRejected  PipelineJobStage = "rejected"
)

// NormalizePipelineJobStage converts UI-facing labels and mixed-case values
// into the canonical backend enum values.
func NormalizePipelineJobStage(raw string) PipelineJobStage {
	stage := strings.ToLower(strings.TrimSpace(raw))
	switch stage {
	case "", string(StageWishlist):
		return StageWishlist
	case string(StageApplied):
		return StageApplied
	case string(StageInterview):
		return StageInterview
	case string(StageOffer), "accepted":
		return StageOffer
	case string(StageRejected):
		return StageRejected
	default:
		return PipelineJobStage(stage)
	}
}

// TimelineEventType classifies a timeline entry.
type TimelineEventType string

const (
	TimelineApplied            TimelineEventType = "applied"
	TimelineFollowUpSent       TimelineEventType = "follow_up_sent"
	TimelineInterviewScheduled TimelineEventType = "interview_scheduled"
	TimelineGhosted            TimelineEventType = "ghosted"
	TimelineStageChanged       TimelineEventType = "stage_changed"
	TimelineNote               TimelineEventType = "note"
)

// TimelineEvent represents one entry in a job application's timeline.
type TimelineEvent struct {
	ID        string
	Type      TimelineEventType
	Label     string
	Detail    string
	CreatedAt time.Time
}

// PipelineContact represents a person associated with a job application.
type PipelineContact struct {
	ID          string
	JobID       string
	UserID      string
	Name        string
	Role        string
	LinkedinURL string
	Email       string
	Notes       string
}

// PipelineJob represents a tracked job application in the user's pipeline.
type PipelineJob struct {
	ID                string
	UserID            string
	CompanyName       string
	JobTitle          string
	Location          string
	Stage             PipelineJobStage
	ResumeID          string
	OptimizedResumeID string
	AtsScore          int
	MatchedKeywords   []string
	MissingKeywords   []string
	JobDescription    string
	JobURL            string
	IsGhosted         bool
	IsArchived        bool
	InterviewAt       *time.Time
	InterviewType     string
	Timeline          []TimelineEvent
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
