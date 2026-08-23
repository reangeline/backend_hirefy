package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

// Key layout:
//   PK = USER#<userId>
//   SK = CONTACT#<jobId>#<contactId>

type contactRepositoryImpl struct {
	client *Client
}

func NewContactRepository(client *Client) outbound.ContactRepository {
	return &contactRepositoryImpl{client: client}
}

type contactItem struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	Type        string `dynamodbav:"Type"`
	ID          string `dynamodbav:"ID"`
	JobID       string `dynamodbav:"JobID"`
	UserID      string `dynamodbav:"UserID"`
	Name        string `dynamodbav:"Name"`
	Role        string `dynamodbav:"Role,omitempty"`
	LinkedinURL string `dynamodbav:"LinkedinURL,omitempty"`
	Email       string `dynamodbav:"Email,omitempty"`
	Notes       string `dynamodbav:"Notes,omitempty"`
	CreatedAt   string `dynamodbav:"CreatedAt"`
}

func (r *contactRepositoryImpl) List(ctx context.Context, userID, jobID string) ([]domain.PipelineContact, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("CONTACT#%s#", jobID)},
		},
	})
	if err != nil {
		return nil, err
	}

	contacts := make([]domain.PipelineContact, 0, len(result.Items))
	for _, av := range result.Items {
		var item contactItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		contacts = append(contacts, domain.PipelineContact{
			ID:          item.ID,
			JobID:       item.JobID,
			UserID:      item.UserID,
			Name:        item.Name,
			Role:        item.Role,
			LinkedinURL: item.LinkedinURL,
			Email:       item.Email,
			Notes:       item.Notes,
		})
	}
	return contacts, nil
}

func (r *contactRepositoryImpl) Add(ctx context.Context, contact *domain.PipelineContact) error {
	if contact.ID == "" {
		contact.ID = uuid.New().String()
	}

	item := contactItem{
		PK:          fmt.Sprintf("USER#%s", contact.UserID),
		SK:          fmt.Sprintf("CONTACT#%s#%s", contact.JobID, contact.ID),
		Type:        "CONTACT",
		ID:          contact.ID,
		JobID:       contact.JobID,
		UserID:      contact.UserID,
		Name:        contact.Name,
		Role:        contact.Role,
		LinkedinURL: contact.LinkedinURL,
		Email:       contact.Email,
		Notes:       contact.Notes,
		CreatedAt:   time.Now().Format(time.RFC3339),
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

func (r *contactRepositoryImpl) Delete(ctx context.Context, userID, jobID, contactID string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("CONTACT#%s#%s", jobID, contactID)},
		},
	})
	return err
}
