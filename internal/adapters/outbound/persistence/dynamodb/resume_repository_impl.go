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

type resumeRepositoryImpl struct {
	client *Client
}

func NewResumeRepository(client *Client) outbound.ResumeRepository {
	return &resumeRepositoryImpl{client: client}
}

// ResumeItem representa um currículo no DynamoDB
type ResumeItem struct {
	PK              string                 `dynamodbav:"PK"`   // USER#<user_id>
	SK              string                 `dynamodbav:"SK"`   // RESUME#<resume_id>
	Type            string                 `dynamodbav:"Type"` // RESUME
	ID              string                 `dynamodbav:"ID"`
	UserID          string                 `dynamodbav:"UserID"`
	OriginalContent string                 `dynamodbav:"OriginalContent"`
	OriginalS3Key   string                 `dynamodbav:"OriginalS3Key,omitempty"`
	ParsedData      map[string]interface{} `dynamodbav:"ParsedData"`
	CreatedAt       string                 `dynamodbav:"CreatedAt"`
	UpdatedAt       string                 `dynamodbav:"UpdatedAt"`
}

// OptimizedResumeItem representa um currículo otimizado no DynamoDB
type OptimizedResumeItem struct {
	PK               string   `dynamodbav:"PK"`   // USER#<user_id>
	SK               string   `dynamodbav:"SK"`   // OPTIMIZED#<optimized_id>
	Type             string   `dynamodbav:"Type"` // OPTIMIZED
	ID               string   `dynamodbav:"ID"`
	UserID           string   `dynamodbav:"UserID"`
	ResumeID         string   `dynamodbav:"ResumeID"`
	JobDescriptionID string   `dynamodbav:"JobDescriptionID"`
	OptimizedContent string   `dynamodbav:"OptimizedContent"`
	OptimizedS3Key   string   `dynamodbav:"OptimizedS3Key,omitempty"`
	MatchScore       float64  `dynamodbav:"MatchScore"`
	Suggestions      []string `dynamodbav:"Suggestions"`
	CreatedAt        string   `dynamodbav:"CreatedAt"`
}

func (r *resumeRepositoryImpl) CreateResume(ctx context.Context, resume *domain.Resume) error {
	item := ResumeItem{
		PK:              fmt.Sprintf("USER#%s", resume.UserID),
		SK:              fmt.Sprintf("RESUME#%s", resume.ID),
		Type:            "RESUME",
		ID:              resume.ID,
		UserID:          resume.UserID,
		OriginalContent: resume.OriginalContent,
		OriginalS3Key:   resume.OriginalS3Key,
		ParsedData:      resume.ParsedData,
		CreatedAt:       resume.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       resume.UpdatedAt.Format(time.RFC3339),
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

func (r *resumeRepositoryImpl) GetResume(ctx context.Context, resumeID string) (*domain.Resume, error) {
	// Como não sabemos o UserID, precisamos fazer um scan ou usar GSI
	// Por simplicidade, vamos assumir que sempre chamamos com ListResumesByUserID
	return nil, fmt.Errorf("use ListResumesByUserID instead")
}

func (r *resumeRepositoryImpl) ListResumesByUserID(ctx context.Context, userID string) ([]*domain.Resume, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: "RESUME#"},
		},
	})

	if err != nil {
		return nil, err
	}

	resumes := make([]*domain.Resume, 0, len(result.Items))
	for _, item := range result.Items {
		var resumeItem ResumeItem
		if err := attributevalue.UnmarshalMap(item, &resumeItem); err != nil {
			continue
		}

		resume, err := r.itemToResume(&resumeItem)
		if err != nil {
			continue
		}

		resumes = append(resumes, resume)
	}

	return resumes, nil
}

func (r *resumeRepositoryImpl) CreateOptimizedResume(ctx context.Context, optimized *domain.OptimizedResume) error {
	item := OptimizedResumeItem{
		PK:               fmt.Sprintf("USER#%s", optimized.UserID),
		SK:               fmt.Sprintf("OPTIMIZED#%s", optimized.ID),
		Type:             "OPTIMIZED",
		ID:               optimized.ID,
		UserID:           optimized.UserID,
		ResumeID:         optimized.ResumeID,
		JobDescriptionID: optimized.JobDescriptionID,
		OptimizedContent: optimized.OptimizedContent,
		OptimizedS3Key:   optimized.OptimizedS3Key,
		MatchScore:       optimized.MatchScore,
		Suggestions:      optimized.Suggestions,
		CreatedAt:        optimized.CreatedAt.Format(time.RFC3339),
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

func (r *resumeRepositoryImpl) GetOptimizedResume(ctx context.Context, optimizedID string) (*domain.OptimizedResume, error) {
	return nil, fmt.Errorf("use ListOptimizedResumesByUserID instead")
}

func (r *resumeRepositoryImpl) ListOptimizedResumesByUserID(ctx context.Context, userID string) ([]*domain.OptimizedResume, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: "OPTIMIZED#"},
		},
	})

	if err != nil {
		return nil, err
	}

	optimizedResumes := make([]*domain.OptimizedResume, 0, len(result.Items))
	for _, item := range result.Items {
		var optimizedItem OptimizedResumeItem
		if err := attributevalue.UnmarshalMap(item, &optimizedItem); err != nil {
			continue
		}

		optimized, err := r.itemToOptimizedResume(&optimizedItem)
		if err != nil {
			continue
		}

		optimizedResumes = append(optimizedResumes, optimized)
	}

	return optimizedResumes, nil
}

func (r *resumeRepositoryImpl) itemToResume(item *ResumeItem) (*domain.Resume, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.Resume{
		ID:              item.ID,
		UserID:          item.UserID,
		OriginalContent: item.OriginalContent,
		OriginalS3Key:   item.OriginalS3Key,
		ParsedData:      item.ParsedData,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

func (r *resumeRepositoryImpl) itemToOptimizedResume(item *OptimizedResumeItem) (*domain.OptimizedResume, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.OptimizedResume{
		ID:               item.ID,
		UserID:           item.UserID,
		ResumeID:         item.ResumeID,
		JobDescriptionID: item.JobDescriptionID,
		OptimizedContent: item.OptimizedContent,
		OptimizedS3Key:   item.OptimizedS3Key,
		MatchScore:       item.MatchScore,
		Suggestions:      item.Suggestions,
		CreatedAt:        createdAt,
	}, nil
}
