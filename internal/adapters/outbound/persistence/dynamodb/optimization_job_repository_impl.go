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

type optimizationJobRepositoryImpl struct {
	client *Client
}

// OptimizationJobItem represents a job record in DynamoDB.
type OptimizationJobItem struct {
	PK                string `dynamodbav:"PK"` // USER#<user_id>
	SK                string `dynamodbav:"SK"` // JOB#<job_id>
	Type              string `dynamodbav:"Type"`
	ID                string `dynamodbav:"ID"`
	UserID            string `dynamodbav:"UserID"`
	ResumeID          string `dynamodbav:"ResumeID"`
	JobDescription    string `dynamodbav:"JobDescription"`
	Status            string `dynamodbav:"Status"`
	Error             string `dynamodbav:"Error,omitempty"`
	OptimizedResumeID string `dynamodbav:"OptimizedResumeID,omitempty"`
	CreatedAt         string `dynamodbav:"CreatedAt"`
	UpdatedAt         string `dynamodbav:"UpdatedAt"`
}

func NewOptimizationJobRepository(client *Client) outbound.OptimizationJobRepository {
	return &optimizationJobRepositoryImpl{client: client}
}

func (r *optimizationJobRepositoryImpl) Create(ctx context.Context, job *domain.OptimizationJob) error {
	item := OptimizationJobItem{
		PK:             fmt.Sprintf("USER#%s", job.UserID),
		SK:             fmt.Sprintf("JOB#%s", job.ID),
		Type:           "JOB",
		ID:             job.ID,
		UserID:         job.UserID,
		ResumeID:       job.ResumeID,
		JobDescription: job.JobDescription,
		Status:         string(job.Status),
		Error:          job.Error,
		CreatedAt:      job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      job.UpdatedAt.Format(time.RFC3339),
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

func (r *optimizationJobRepositoryImpl) UpdateStatus(ctx context.Context, userID, jobID string, status domain.OptimizationJobStatus, errMsg, optimizedResumeID string) error {
	pk := fmt.Sprintf("USER#%s", userID)
	sk := fmt.Sprintf("JOB#%s", jobID)

	update := "SET #status = :status, #updatedAt = :updatedAt"
	values := map[string]types.AttributeValue{
		":status":    &types.AttributeValueMemberS{Value: string(status)},
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	exprAttrNames := map[string]string{
		"#status":    "Status",
		"#updatedAt": "UpdatedAt",
	}

	if errMsg != "" {
		update += ", #error = :error"
		values[":error"] = &types.AttributeValueMemberS{Value: errMsg}
		exprAttrNames["#error"] = "Error"
	}

	if optimizedResumeID != "" {
		update += ", #optimizedResumeId = :optimizedResumeId"
		values[":optimizedResumeId"] = &types.AttributeValueMemberS{Value: optimizedResumeID}
		exprAttrNames["#optimizedResumeId"] = "OptimizedResumeID"
	}

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          aws.String(update),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: values,
	})

	return err
}

func (r *optimizationJobRepositoryImpl) Get(ctx context.Context, userID, jobID string) (*domain.OptimizationJob, error) {
	pk := fmt.Sprintf("USER#%s", userID)
	sk := fmt.Sprintf("JOB#%s", jobID)

	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrJobNotFound
	}

	var item OptimizationJobItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	return r.itemToDomain(&item)
}

func (r *optimizationJobRepositoryImpl) itemToDomain(item *OptimizationJobItem) (*domain.OptimizationJob, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.OptimizationJob{
		ID:                item.ID,
		UserID:            item.UserID,
		ResumeID:          item.ResumeID,
		JobDescription:    item.JobDescription,
		Status:            domain.OptimizationJobStatus(item.Status),
		Error:             item.Error,
		OptimizedResumeID: item.OptimizedResumeID,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}
