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

type linkedInPostRepositoryImpl struct {
	client *Client
}

func NewLinkedInPostRepository(client *Client) outbound.LinkedInPostIdeasRepository {
	return &linkedInPostRepositoryImpl{client: client}
}

// Singleton por usuário — PK USER#<user_id>, SK fixo LINKEDINPOSTIDEAS (sem histórico,
// spec 018).
const linkedInPostIdeasSK = "LINKEDINPOSTIDEAS"

type LinkedInPostTopicItem struct {
	Title string `dynamodbav:"title"`
	Angle string `dynamodbav:"angle"`
}

type LinkedInPostIdeasItem struct {
	PK        string                  `dynamodbav:"PK"`
	SK        string                  `dynamodbav:"SK"`
	Type      string                  `dynamodbav:"Type"`
	ID        string                  `dynamodbav:"ID"`
	UserID    string                  `dynamodbav:"UserID"`
	Topics    []LinkedInPostTopicItem `dynamodbav:"Topics"`
	CreatedAt string                  `dynamodbav:"CreatedAt"`
	UpdatedAt string                  `dynamodbav:"UpdatedAt"`
}

func (r *linkedInPostRepositoryImpl) Upsert(ctx context.Context, ideas *domain.LinkedInPostIdeas) error {
	topics := make([]LinkedInPostTopicItem, 0, len(ideas.Topics))
	for _, t := range ideas.Topics {
		topics = append(topics, LinkedInPostTopicItem{Title: t.Title, Angle: t.Angle})
	}

	item := LinkedInPostIdeasItem{
		PK:        fmt.Sprintf("USER#%s", ideas.UserID),
		SK:        linkedInPostIdeasSK,
		Type:      "LINKEDIN_POST_IDEAS",
		ID:        ideas.ID,
		UserID:    ideas.UserID,
		Topics:    topics,
		CreatedAt: ideas.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ideas.UpdatedAt.Format(time.RFC3339),
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

func (r *linkedInPostRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*domain.LinkedInPostIdeas, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: linkedInPostIdeasSK},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrLinkedInPostIdeasNotFound
	}

	var item LinkedInPostIdeasItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	return itemToLinkedInPostIdeas(&item)
}

func itemToLinkedInPostIdeas(item *LinkedInPostIdeasItem) (*domain.LinkedInPostIdeas, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	topics := make([]domain.LinkedInPostTopic, 0, len(item.Topics))
	for _, t := range item.Topics {
		topics = append(topics, domain.LinkedInPostTopic{Title: t.Title, Angle: t.Angle})
	}

	return &domain.LinkedInPostIdeas{
		ID:        item.ID,
		UserID:    item.UserID,
		Topics:    topics,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
