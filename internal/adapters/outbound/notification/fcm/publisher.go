package fcm

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
	"google.golang.org/api/option"
)

type Publisher struct {
	client   *messaging.Client
	userRepo outbound.UserRepository
}

// NewPublisher cria um NotificationPublisher usando Firebase Cloud Messaging.
// Se credentialsFile estiver vazio, retorna nil para que o chamador possa usar noop.
// projectID é opcional, mas necessário se o JSON não incluir project_id.
func NewPublisher(ctx context.Context, credentialsFile string, projectID string, userRepo outbound.UserRepository) (outbound.NotificationPublisher, error) {
	if credentialsFile == "" {
		log.Printf("[fcm] skipping init: FIREBASE_CREDENTIALS_FILE is empty")
		return nil, nil
	}

	cfg := &firebase.Config{}
	if projectID != "" {
		cfg.ProjectID = projectID
	}

	app, err := firebase.NewApp(ctx, cfg, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to init firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init firebase messaging client: %w", err)
	}

	return &Publisher{client: client, userRepo: userRepo}, nil
}

func (p *Publisher) NotifyResumeOptimized(ctx context.Context, userID, jobID, optimizedResumeID string) error {
	if p == nil || p.client == nil {
		return nil
	}

	user, err := p.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.FCMToken == "" {
		log.Printf("[fcm] user %s has no FCM token; skipping push", userID)
		return nil
	}

	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg := &messaging.Message{
		Token: user.FCMToken,
		Notification: &messaging.Notification{
			Title: "Resume optimized",
			Body:  "Your optimized resume is ready.",
		},
		Data: map[string]string{
			"type":                "resume_optimized",
			"job_id":              jobID,
			"optimized_resume_id": optimizedResumeID,
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default"}},
		},
		Android: &messaging.AndroidConfig{Priority: "high"},
	}

	if _, err := p.client.Send(sendCtx, msg); err != nil {
		return fmt.Errorf("failed to send fcm notification: %w", err)
	}

	return nil
}
