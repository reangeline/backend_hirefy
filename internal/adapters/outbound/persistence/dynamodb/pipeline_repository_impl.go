package dynamodb

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/reangeline/backend_applywise/internal/core/domain"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type pipelineRepositoryImpl struct {
	client *Client
}

func NewPipelineRepository(client *Client) outbound.PipelineRepository {
	return &pipelineRepositoryImpl{client: client}
}

// TimelineEventItem is the DynamoDB representation of a timeline event.
type TimelineEventItem struct {
	ID        string `dynamodbav:"ID"`
	Type      string `dynamodbav:"Type"`
	Label     string `dynamodbav:"Label"`
	Detail    string `dynamodbav:"Detail"`
	CreatedAt string `dynamodbav:"CreatedAt"`
}

// PipelineJobItem is the DynamoDB record for a pipeline job.
type PipelineJobItem struct {
	PK                string              `dynamodbav:"PK"`
	SK                string              `dynamodbav:"SK"`
	GSI1PK            string              `dynamodbav:"GSI1PK"`
	GSI1SK            string              `dynamodbav:"GSI1SK"`
	Type              string              `dynamodbav:"Type"`
	ID                string              `dynamodbav:"ID"`
	UserID            string              `dynamodbav:"UserID"`
	CompanyName       string              `dynamodbav:"CompanyName"`
	JobTitle          string              `dynamodbav:"JobTitle"`
	Location          string              `dynamodbav:"Location,omitempty"`
	Stage             string              `dynamodbav:"Stage"`
	ResumeID          string              `dynamodbav:"ResumeID,omitempty"`
	OptimizedResumeID string              `dynamodbav:"OptimizedResumeID,omitempty"`
	AtsScore          int                 `dynamodbav:"AtsScore,omitempty"`
	MatchedKeywords   []string            `dynamodbav:"MatchedKeywords,omitempty"`
	MissingKeywords   []string            `dynamodbav:"MissingKeywords,omitempty"`
	JobDescription    string              `dynamodbav:"JobDescription,omitempty"`
	JobURL            string              `dynamodbav:"JobURL,omitempty"`
	IsGhosted         bool                `dynamodbav:"IsGhosted"`
	IsArchived        bool                `dynamodbav:"IsArchived"`
	InterviewAt       *string             `dynamodbav:"InterviewAt,omitempty"`
	InterviewType     string              `dynamodbav:"InterviewType,omitempty"`
	Timeline          []TimelineEventItem `dynamodbav:"Timeline,omitempty"`
	CreatedAt         string              `dynamodbav:"CreatedAt"`
	UpdatedAt         string              `dynamodbav:"UpdatedAt"`
}

func (r *pipelineRepositoryImpl) Create(ctx context.Context, job *domain.PipelineJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}

	item := r.domainToItem(job)
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

func (r *pipelineRepositoryImpl) Get(ctx context.Context, userID, jobID string) (*domain.PipelineJob, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PIPELINE#%s", jobID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, domain.ErrPipelineJobNotFound
	}

	var item PipelineJobItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}
	return r.itemToDomain(&item)
}

func (r *pipelineRepositoryImpl) List(ctx context.Context, userID string) ([]domain.PipelineJob, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: "PIPELINE#"},
		},
	})
	if err != nil {
		return nil, err
	}

	jobs := make([]domain.PipelineJob, 0, len(result.Items))
	for _, av := range result.Items {
		var item PipelineJobItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		job, err := r.itemToDomain(&item)
		if err != nil {
			continue
		}
		jobs = append(jobs, *job)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs, nil
}

func (r *pipelineRepositoryImpl) Update(ctx context.Context, job *domain.PipelineJob) error {
	job.UpdatedAt = time.Now()
	item := r.domainToItem(job)
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

func (r *pipelineRepositoryImpl) Delete(ctx context.Context, userID, jobID string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PIPELINE#%s", jobID)},
		},
	})
	return err
}

func (r *pipelineRepositoryImpl) AppendTimelineEvent(ctx context.Context, userID, jobID string, event domain.TimelineEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	eventItem := TimelineEventItem{
		ID:        event.ID,
		Type:      string(event.Type),
		Label:     event.Label,
		Detail:    event.Detail,
		CreatedAt: event.CreatedAt.Format(time.RFC3339),
	}

	eventAV, err := attributevalue.MarshalMap(eventItem)
	if err != nil {
		return err
	}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("PIPELINE#%s", jobID)},
		},
		UpdateExpression: aws.String("SET #timeline = list_append(if_not_exists(#timeline, :empty), :event), UpdatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#timeline": "Timeline",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":event":     &types.AttributeValueMemberL{Value: []types.AttributeValue{&types.AttributeValueMemberM{Value: eventAV}}},
			":empty":     &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			":updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	return err
}

func (r *pipelineRepositoryImpl) domainToItem(job *domain.PipelineJob) *PipelineJobItem {
	item := &PipelineJobItem{
		PK:                fmt.Sprintf("USER#%s", job.UserID),
		SK:                fmt.Sprintf("PIPELINE#%s", job.ID),
		GSI1PK:            fmt.Sprintf("USER#%s", job.UserID),
		GSI1SK:            fmt.Sprintf("PIPELINE#%s#%s", domain.NormalizePipelineJobStage(string(job.Stage)), job.CreatedAt.Format(time.RFC3339)),
		Type:              "PIPELINE",
		ID:                job.ID,
		UserID:            job.UserID,
		CompanyName:       job.CompanyName,
		JobTitle:          job.JobTitle,
		Location:          job.Location,
		Stage:             string(domain.NormalizePipelineJobStage(string(job.Stage))),
		ResumeID:          job.ResumeID,
		OptimizedResumeID: job.OptimizedResumeID,
		AtsScore:          job.AtsScore,
		MatchedKeywords:   job.MatchedKeywords,
		MissingKeywords:   job.MissingKeywords,
		JobDescription:    job.JobDescription,
		JobURL:            job.JobURL,
		IsGhosted:         job.IsGhosted,
		IsArchived:        job.IsArchived,
		InterviewType:     job.InterviewType,
		CreatedAt:         job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         job.UpdatedAt.Format(time.RFC3339),
	}

	if job.InterviewAt != nil {
		s := job.InterviewAt.Format(time.RFC3339)
		item.InterviewAt = &s
	}

	if len(job.Timeline) > 0 {
		item.Timeline = make([]TimelineEventItem, len(job.Timeline))
		for i, e := range job.Timeline {
			item.Timeline[i] = TimelineEventItem{
				ID:        e.ID,
				Type:      string(e.Type),
				Label:     e.Label,
				Detail:    e.Detail,
				CreatedAt: e.CreatedAt.Format(time.RFC3339),
			}
		}
	}

	return item
}

func (r *pipelineRepositoryImpl) itemToDomain(item *PipelineJobItem) (*domain.PipelineJob, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		createdAt = time.Time{}
	}
	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		updatedAt = time.Time{}
	}

	job := &domain.PipelineJob{
		ID:                item.ID,
		UserID:            item.UserID,
		CompanyName:       item.CompanyName,
		JobTitle:          item.JobTitle,
		Location:          item.Location,
		Stage:             domain.NormalizePipelineJobStage(item.Stage),
		ResumeID:          item.ResumeID,
		OptimizedResumeID: item.OptimizedResumeID,
		AtsScore:          item.AtsScore,
		MatchedKeywords:   item.MatchedKeywords,
		MissingKeywords:   item.MissingKeywords,
		JobDescription:    item.JobDescription,
		JobURL:            item.JobURL,
		IsGhosted:         item.IsGhosted,
		IsArchived:        item.IsArchived,
		InterviewType:     item.InterviewType,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	if item.InterviewAt != nil {
		t, err := time.Parse(time.RFC3339, *item.InterviewAt)
		if err == nil {
			job.InterviewAt = &t
		}
	}

	if len(item.Timeline) > 0 {
		job.Timeline = make([]domain.TimelineEvent, len(item.Timeline))
		for i, e := range item.Timeline {
			evt := domain.TimelineEvent{
				ID:    e.ID,
				Type:  domain.TimelineEventType(e.Type),
				Label: e.Label,
				Detail: e.Detail,
			}
			if t, err := time.Parse(time.RFC3339, e.CreatedAt); err == nil {
				evt.CreatedAt = t
			}
			job.Timeline[i] = evt
		}
	}

	return job, nil
}

