package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type coachSuggestionRepositoryImpl struct {
	client *Client
}

func NewCoachSuggestionRepository(client *Client) outbound.CoachSuggestionRepository {
	return &coachSuggestionRepositoryImpl{client: client}
}

// Singleton por (usuário, vaga, estágio) — PK USER#<user_id>, SK
// JOB#<job_id>#COACH#<stage> (sem histórico, mesmo padrão de LinkedInPostIdeas/LinkedInScan).
func coachSuggestionSK(jobID string, stage domain.PipelineJobStage) string {
	return fmt.Sprintf("JOB#%s#COACH#%s", jobID, stage)
}

type CoachSuggestionItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	Type      string `dynamodbav:"Type"`
	UserID    string `dynamodbav:"UserID"`
	JobID     string `dynamodbav:"JobID"`
	Stage     string `dynamodbav:"Stage"`
	Content   string `dynamodbav:"Content"`
	CoachType string `dynamodbav:"CoachType"`
	CreatedAt string `dynamodbav:"CreatedAt"`
	UpdatedAt string `dynamodbav:"UpdatedAt"`
}

func (r *coachSuggestionRepositoryImpl) Upsert(ctx context.Context, s *domain.CoachSuggestion) error {
	item := CoachSuggestionItem{
		PK:        fmt.Sprintf("USER#%s", s.UserID),
		SK:        coachSuggestionSK(s.JobID, s.Stage),
		Type:      "COACH_SUGGESTION",
		UserID:    s.UserID,
		JobID:     s.JobID,
		Stage:     string(s.Stage),
		Content:   s.Content,
		CoachType: s.Type,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.tableName),
		Item:      av,
	})

	return err
}

func (r *coachSuggestionRepositoryImpl) Get(ctx context.Context, userID, jobID string, stage domain.PipelineJobStage) (*domain.CoachSuggestion, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: coachSuggestionSK(jobID, stage)},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrCoachSuggestionNotFound
	}

	var item CoachSuggestionItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.CoachSuggestion{
		UserID:    item.UserID,
		JobID:     item.JobID,
		Stage:     domain.PipelineJobStage(item.Stage),
		Content:   item.Content,
		Type:      item.CoachType,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
