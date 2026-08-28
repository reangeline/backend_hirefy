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
//   SK = INTERVIEW#<jobId>#<questionId>

type interviewRepositoryImpl struct {
	client *Client
}

func NewInterviewRepository(client *Client) outbound.InterviewRepository {
	return &interviewRepositoryImpl{client: client}
}

type interviewQuestionItem struct {
	PK            string   `dynamodbav:"PK"`
	SK            string   `dynamodbav:"SK"`
	Type          string   `dynamodbav:"Type"`
	ID            string   `dynamodbav:"ID"`
	JobID         string   `dynamodbav:"JobID"`
	UserID        string   `dynamodbav:"UserID"`
	Kind          string   `dynamodbav:"Kind"`
	Question      string   `dynamodbav:"Question"`
	WhatTheyWant  string   `dynamodbav:"WhatTheyWant,omitempty"`
	MethodHint    string   `dynamodbav:"MethodHint,omitempty"`
	Answer        string   `dynamodbav:"Answer,omitempty"`
	ContentScore  int      `dynamodbav:"ContentScore"`
	STARSituation int      `dynamodbav:"STARSituation"`
	STARTask      int      `dynamodbav:"STARTask"`
	STARAction    int      `dynamodbav:"STARAction"`
	STARResult    int      `dynamodbav:"STARResult"`
	Strengths     []string `dynamodbav:"Strengths,omitempty"`
	Gaps          []string `dynamodbav:"Gaps,omitempty"`
	ModelAnswer   string   `dynamodbav:"ModelAnswer,omitempty"`
	FollowUp      string   `dynamodbav:"FollowUp,omitempty"`
	CreatedAt     string   `dynamodbav:"CreatedAt"`
	AnsweredAt    string   `dynamodbav:"AnsweredAt,omitempty"`
}

func itemToInterviewQuestion(item *interviewQuestionItem) (*domain.InterviewQuestion, error) {
	q := &domain.InterviewQuestion{
		ID:            item.ID,
		JobID:         item.JobID,
		UserID:        item.UserID,
		Kind:          domain.InterviewQuestionKind(item.Kind),
		Question:      item.Question,
		WhatTheyWant:  item.WhatTheyWant,
		MethodHint:    item.MethodHint,
		Answer:        item.Answer,
		ContentScore:  item.ContentScore,
		STARSituation: item.STARSituation,
		STARTask:      item.STARTask,
		STARAction:    item.STARAction,
		STARResult:    item.STARResult,
		Strengths:     item.Strengths,
		Gaps:          item.Gaps,
		ModelAnswer:   item.ModelAnswer,
		FollowUp:      item.FollowUp,
	}
	if item.CreatedAt != "" {
		t, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err == nil {
			q.CreatedAt = t
		}
	}
	if item.AnsweredAt != "" {
		t, err := time.Parse(time.RFC3339, item.AnsweredAt)
		if err == nil {
			q.AnsweredAt = &t
		}
	}
	return q, nil
}

func interviewQuestionToItem(q *domain.InterviewQuestion) interviewQuestionItem {
	item := interviewQuestionItem{
		PK:            fmt.Sprintf("USER#%s", q.UserID),
		SK:            fmt.Sprintf("INTERVIEW#%s#%s", q.JobID, q.ID),
		Type:          "INTERVIEW_QUESTION",
		ID:            q.ID,
		JobID:         q.JobID,
		UserID:        q.UserID,
		Kind:          string(q.Kind),
		Question:      q.Question,
		WhatTheyWant:  q.WhatTheyWant,
		MethodHint:    q.MethodHint,
		Answer:        q.Answer,
		ContentScore:  q.ContentScore,
		STARSituation: q.STARSituation,
		STARTask:      q.STARTask,
		STARAction:    q.STARAction,
		STARResult:    q.STARResult,
		Strengths:     q.Strengths,
		Gaps:          q.Gaps,
		ModelAnswer:   q.ModelAnswer,
		FollowUp:      q.FollowUp,
		CreatedAt:     q.CreatedAt.Format(time.RFC3339),
	}
	if q.AnsweredAt != nil {
		item.AnsweredAt = q.AnsweredAt.Format(time.RFC3339)
	}
	return item
}

func (r *interviewRepositoryImpl) List(ctx context.Context, userID, jobID string) ([]domain.InterviewQuestion, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("INTERVIEW#%s#", jobID)},
		},
	})
	if err != nil {
		return nil, err
	}

	questions := make([]domain.InterviewQuestion, 0, len(result.Items))
	for _, av := range result.Items {
		var item interviewQuestionItem
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			continue
		}
		q, err := itemToInterviewQuestion(&item)
		if err != nil {
			continue
		}
		questions = append(questions, *q)
	}
	return questions, nil
}

func (r *interviewRepositoryImpl) Get(ctx context.Context, userID, jobID, questionID string) (*domain.InterviewQuestion, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("INTERVIEW#%s#%s", jobID, questionID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, domain.ErrInterviewQuestionNotFound
	}

	var item interviewQuestionItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}
	return itemToInterviewQuestion(&item)
}

func (r *interviewRepositoryImpl) Create(ctx context.Context, question *domain.InterviewQuestion) error {
	if question.ID == "" {
		question.ID = uuid.New().String()
	}
	if question.CreatedAt.IsZero() {
		question.CreatedAt = time.Now()
	}

	av, err := attributevalue.MarshalMap(interviewQuestionToItem(question))
	if err != nil {
		return err
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.tableName),
		Item:      av,
	})
	return err
}

func (r *interviewRepositoryImpl) Update(ctx context.Context, question *domain.InterviewQuestion) error {
	av, err := attributevalue.MarshalMap(interviewQuestionToItem(question))
	if err != nil {
		return err
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.client.tableName),
		Item:      av,
	})
	return err
}
