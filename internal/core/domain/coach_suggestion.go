package domain

import "time"

// CoachSuggestion is the most recent AI coaching content generated for a pipeline job at a
// given stage — cached so revisiting the Coach tab doesn't spend another credit re-asking
// the AI for the same thing. One record per (job, stage), no history: a new generation (or
// an explicit "regenerate") overwrites the previous one, same pattern as LinkedInPostIdeas
// and LinkedInScan.
type CoachSuggestion struct {
	UserID    string
	JobID     string
	Stage     PipelineJobStage
	Content   string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
