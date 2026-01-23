package stripe

import (
	"encoding/json"
	"fmt"
)

// WebhookEvent representa um evento do Stripe
type WebhookEvent struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
	Created int64                  `json:"created"`
}

// ParseWebhookEvent faz parse de um evento de webhook
func ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}
	return &event, nil
}

// GetEventType retorna o tipo do evento
func (e *WebhookEvent) GetEventType() string {
	return e.Type
}

// GetEventData retorna os dados do evento
func (e *WebhookEvent) GetEventData() map[string]interface{} {
	return e.Data
}
