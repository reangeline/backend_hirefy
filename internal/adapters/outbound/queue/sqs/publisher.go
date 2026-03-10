package sqs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type sqsPublisher struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSPublisher(cfg aws.Config, queueURL string) outbound.QueuePublisher {
	return &sqsPublisher{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}
}

func (p *sqsPublisher) PublishOptimizationJob(ctx context.Context, msg outbound.OptimizationJobMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(payload)),
	})

	return err
}
