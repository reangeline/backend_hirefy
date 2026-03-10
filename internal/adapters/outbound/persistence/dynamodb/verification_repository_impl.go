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

type verificationRepositoryImpl struct {
	client *Client
}

func NewVerificationRepository(client *Client) outbound.VerificationRepository {
	return &verificationRepositoryImpl{client: client}
}

type VerificationCodeItem struct {
	PK        string `dynamodbav:"PK"`   // VERIFICATION#<email>
	SK        string `dynamodbav:"SK"`   // CODE
	Type      string `dynamodbav:"Type"` // VERIFICATION
	Email     string `dynamodbav:"Email"`
	Code      string `dynamodbav:"Code"`
	TTL       int64  `dynamodbav:"TTL"` // Auto-delete após expirar
	ExpiresAt string `dynamodbav:"ExpiresAt"`
	CreatedAt string `dynamodbav:"CreatedAt"`
}

func (r *verificationRepositoryImpl) Create(ctx context.Context, code *domain.VerificationCode) error {
	item := VerificationCodeItem{
		PK:        fmt.Sprintf("VERIFICATION#%s", code.Email),
		SK:        "CODE",
		Type:      "VERIFICATION",
		Email:     code.Email,
		Code:      code.Code,
		TTL:       code.ExpiresAt.Unix(), // DynamoDB auto-deleta
		ExpiresAt: code.ExpiresAt.Format(time.RFC3339),
		CreatedAt: code.CreatedAt.Format(time.RFC3339),
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

func (r *verificationRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.VerificationCode, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("VERIFICATION#%s", email)},
			"SK": &types.AttributeValueMemberS{Value: "CODE"},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("verification code not found")
	}

	var item VerificationCodeItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, item.ExpiresAt)
	if err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.VerificationCode{
		Email:     item.Email,
		Code:      item.Code,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}, nil
}

func (r *verificationRepositoryImpl) Delete(ctx context.Context, email string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("VERIFICATION#%s", email)},
			"SK": &types.AttributeValueMemberS{Value: "CODE"},
		},
	})

	return err
}
