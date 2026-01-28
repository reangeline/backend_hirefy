package inbound

import "context"

type RevenueCatService interface {
	ProcessWebhookEvent(ctx context.Context, event map[string]interface{}) error
}
