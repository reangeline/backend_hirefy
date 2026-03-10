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
	ResumeType      string                 `dynamodbav:"ResumeType,omitempty"`
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
	PK                  string                 `dynamodbav:"PK"`   // USER#<user_id>
	SK                  string                 `dynamodbav:"SK"`   // OPTIMIZED#<optimized_id>
	Type                string                 `dynamodbav:"Type"` // OPTIMIZED
	ID                  string                 `dynamodbav:"ID"`
	UserID              string                 `dynamodbav:"UserID"`
	ResumeID            string                 `dynamodbav:"ResumeID"`
	SourceResumeID      string                 `dynamodbav:"SourceResumeID"`
	JobDescriptionID    string                 `dynamodbav:"JobDescriptionID"`
	OptimizedContent    string                 `dynamodbav:"OptimizedContent"`
	OptimizedS3Key      string                 `dynamodbav:"OptimizedS3Key,omitempty"`
	ParsedData          map[string]interface{} `dynamodbav:"ParsedData"`
	MatchScore          float64                `dynamodbav:"MatchScore"`
	Suggestions         []string               `dynamodbav:"Suggestions"`
	MissingRequirements []string               `dynamodbav:"MissingRequirements"`
	SalaryEstimate      *SalaryEstimateItem    `dynamodbav:"SalaryEstimate,omitempty"`
	CreatedAt           string                 `dynamodbav:"CreatedAt"`
}

// SalaryEstimateItem é a representação do DynamoDB para estimativa salarial.
type SalaryEstimateItem struct {
	Found      bool    `dynamodbav:"Found"`
	Currency   string  `dynamodbav:"Currency,omitempty"`
	MinSalary  float64 `dynamodbav:"MinSalary,omitempty"`
	MaxSalary  float64 `dynamodbav:"MaxSalary,omitempty"`
	Midpoint   float64 `dynamodbav:"Midpoint,omitempty"`
	Period     string  `dynamodbav:"Period,omitempty"`
	Location   string  `dynamodbav:"Location,omitempty"`
	Seniority  string  `dynamodbav:"Seniority,omitempty"`
	Notes      string  `dynamodbav:"Notes,omitempty"`
	Disclaimer string  `dynamodbav:"Disclaimer,omitempty"`
}

func (r *resumeRepositoryImpl) CreateResume(ctx context.Context, resume *domain.Resume) error {
	item := ResumeItem{
		PK:              fmt.Sprintf("USER#%s", resume.UserID),
		SK:              fmt.Sprintf("RESUME#%s", resume.ID),
		Type:            "RESUME",
		ResumeType:      resume.Type,
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

func (r *resumeRepositoryImpl) GetResume(ctx context.Context, userID, resumeID string) (*domain.Resume, error) {
	// Usa userID e resumeID para compor PK e SK diretamente (GetItem é muito mais eficiente que Scan)
	pk := fmt.Sprintf("USER#%s", userID)
	sk := fmt.Sprintf("RESUME#%s", resumeID)

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
		return nil, domain.ErrResumeNotFound
	}

	var resumeItem ResumeItem
	if err := attributevalue.UnmarshalMap(result.Item, &resumeItem); err != nil {
		return nil, err
	}

	return r.itemToResume(&resumeItem)
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
		PK:                  fmt.Sprintf("USER#%s", optimized.UserID),
		SK:                  fmt.Sprintf("OPTIMIZED#%s", optimized.ID),
		Type:                "OPTIMIZED",
		ID:                  optimized.ID,
		UserID:              optimized.UserID,
		ResumeID:            optimized.ResumeID,
		SourceResumeID:      optimized.SourceResumeID,
		JobDescriptionID:    optimized.JobDescriptionID,
		OptimizedContent:    optimized.OptimizedContent,
		OptimizedS3Key:      optimized.OptimizedS3Key,
		ParsedData:          optimized.ParsedData,
		MatchScore:          optimized.MatchScore,
		Suggestions:         optimized.Suggestions,
		MissingRequirements: optimized.MissingRequirements,
		CreatedAt:           optimized.CreatedAt.Format(time.RFC3339),
	}

	if optimized.SalaryEstimate != nil {
		item.SalaryEstimate = &SalaryEstimateItem{
			Found:      optimized.SalaryEstimate.Found,
			Currency:   optimized.SalaryEstimate.Currency,
			MinSalary:  optimized.SalaryEstimate.MinSalary,
			MaxSalary:  optimized.SalaryEstimate.MaxSalary,
			Midpoint:   optimized.SalaryEstimate.Midpoint,
			Period:     optimized.SalaryEstimate.Period,
			Location:   optimized.SalaryEstimate.Location,
			Seniority:  optimized.SalaryEstimate.Seniority,
			Notes:      optimized.SalaryEstimate.Notes,
			Disclaimer: optimized.SalaryEstimate.Disclaimer,
		}
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

func (r *resumeRepositoryImpl) UpdateResume(ctx context.Context, resume *domain.Resume) error {
	// Preserve CreatedAt if existing
	existing, err := r.GetResume(ctx, resume.UserID, resume.ID)
	var createdAt time.Time
	if err == nil && existing != nil {
		createdAt = existing.CreatedAt
		// Preserve existing resume type when not provided
		if resume.Type == "" {
			resume.Type = existing.Type
		}
	} else {
		createdAt = resume.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
	}

	if resume.UpdatedAt.IsZero() {
		resume.UpdatedAt = time.Now()
	}

	item := ResumeItem{
		PK:              fmt.Sprintf("USER#%s", resume.UserID),
		SK:              fmt.Sprintf("RESUME#%s", resume.ID),
		Type:            "RESUME",
		ResumeType:      resume.Type,
		ID:              resume.ID,
		UserID:          resume.UserID,
		OriginalContent: resume.OriginalContent,
		OriginalS3Key:   resume.OriginalS3Key,
		ParsedData:      resume.ParsedData,
		CreatedAt:       createdAt.Format(time.RFC3339),
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

func (r *resumeRepositoryImpl) DeleteResume(ctx context.Context, userID, resumeID string) error {
	// Usa userID para compor diretamente PK e SK sem precisar de Scan
	pk := fmt.Sprintf("USER#%s", userID)
	sk := fmt.Sprintf("RESUME#%s", resumeID)

	// Verifica se existe antes de deletar (opcional)
	getResult, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		return err
	}

	if getResult.Item == nil {
		return domain.ErrResumeNotFound
	}

	// Delete usando PK e SK
	_, err = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	return err
}

func (r *resumeRepositoryImpl) DeleteOptimizedResume(ctx context.Context, userID, optimizedID string) error {
	pk := fmt.Sprintf("USER#%s", userID)
	sk := fmt.Sprintf("OPTIMIZED#%s", optimizedID)

	getResult, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	if err != nil {
		return err
	}

	if getResult.Item == nil {
		return domain.ErrResumeNotFound
	}

	_, err = r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})

	return err
}

func (r *resumeRepositoryImpl) UpdateOptimizedResume(ctx context.Context, optimized *domain.OptimizedResume) error {
	pk := fmt.Sprintf("USER#%s", optimized.UserID)
	sk := fmt.Sprintf("OPTIMIZED#%s", optimized.ID)

	getResult, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return err
	}
	if getResult.Item == nil {
		return domain.ErrResumeNotFound
	}

	var existing OptimizedResumeItem
	if err := attributevalue.UnmarshalMap(getResult.Item, &existing); err != nil {
		return err
	}

	// Merge ParsedData — replace only keys provided
	if existing.ParsedData == nil {
		existing.ParsedData = map[string]interface{}{}
	}
	for k, v := range optimized.ParsedData {
		existing.ParsedData[k] = v
	}

	av, err := attributevalue.MarshalMap(existing)
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
		Type:            item.ResumeType,
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

	optimized := &domain.OptimizedResume{
		ID:                  item.ID,
		UserID:              item.UserID,
		ResumeID:            item.ResumeID,
		SourceResumeID:      item.SourceResumeID,
		JobDescriptionID:    item.JobDescriptionID,
		OptimizedContent:    item.OptimizedContent,
		OptimizedS3Key:      item.OptimizedS3Key,
		ParsedData:          item.ParsedData,
		MatchScore:          item.MatchScore,
		Suggestions:         item.Suggestions,
		CreatedAt:           createdAt,
		MissingRequirements: item.MissingRequirements,
	}

	if item.SalaryEstimate != nil {
		optimized.SalaryEstimate = &domain.SalaryEstimate{
			Found:      item.SalaryEstimate.Found,
			Currency:   item.SalaryEstimate.Currency,
			MinSalary:  item.SalaryEstimate.MinSalary,
			MaxSalary:  item.SalaryEstimate.MaxSalary,
			Midpoint:   item.SalaryEstimate.Midpoint,
			Period:     item.SalaryEstimate.Period,
			Location:   item.SalaryEstimate.Location,
			Seniority:  item.SalaryEstimate.Seniority,
			Notes:      item.SalaryEstimate.Notes,
			Disclaimer: item.SalaryEstimate.Disclaimer,
		}
	}

	return optimized, nil
}
