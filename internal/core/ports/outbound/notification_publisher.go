package outbound

import "context"

// NotificationPublisher delivers async notifications (e.g., WebSocket push).
type NotificationPublisher interface {
	NotifyResumeOptimized(ctx context.Context, userID, jobID, optimizedResumeID string) error
}
