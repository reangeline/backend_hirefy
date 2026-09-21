package domain

import "time"

// LinkedInPostTopic is one suggested theme/angle to write a LinkedIn post about — not a
// news item, an evergreen angle tailored to the candidate's profile (spec 018). Draft holds
// the last post text generated for this topic (empty until the user asks for one) — cached
// so revisiting/switching topics doesn't spend another AI call for the same draft.
type LinkedInPostTopic struct {
	Title string `json:"title"`
	Angle string `json:"angle"`
	Draft string `json:"draft,omitempty"`
}

// LinkedInPostIdeas is the most recent batch of topic suggestions generated for a user.
// Only the latest batch is kept per user — no history (same pattern as LinkedInScan).
type LinkedInPostIdeas struct {
	ID        string              `json:"id"`
	UserID    string              `json:"user_id"`
	Topics    []LinkedInPostTopic `json:"topics"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}
