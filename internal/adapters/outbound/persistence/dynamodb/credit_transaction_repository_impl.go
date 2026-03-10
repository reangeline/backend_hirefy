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

type creditTransactionRepositoryImpl struct {
	client *Client
}

func NewCreditTransactionRepository(client *Client) outbound.CreditTransactionRepository {
	return &creditTransactionRepositoryImpl{client: client}
}

type CreditTransactionItem struct {
	PK        string            `dynamodbav:"PK"`   // USER#<user_id>
	SK        string            `dynamodbav:"SK"`   // CREDITTX#<timestamp>#<tx_id>
	Type      string            `dynamodbav:"Type"` // CREDIT_TRANSACTION
	ID        string            `dynamodbav:"ID"`
	UserID    string            `dynamodbav:"UserID"`
	Amount    int               `dynamodbav:"Amount"`
	TxType    string            `dynamodbav:"TxType"`
	Reason    string            `dynamodbav:"Reason"`
	Metadata  map[string]string `dynamodbav:"Metadata,omitempty"`
	CreatedAt string            `dynamodbav:"CreatedAt"`
}

func (r *creditTransactionRepositoryImpl) Create(ctx context.Context, transaction *domain.CreditTransaction) error {
	// SK com timestamp para ordenação: CREDITTX#<timestamp>#<id>
	timestamp := transaction.CreatedAt.Unix()
	sk := fmt.Sprintf("CREDITTX#%d#%s", timestamp, transaction.ID)

	item := CreditTransactionItem{
		PK:        fmt.Sprintf("USER#%s", transaction.UserID),
		SK:        sk,
		Type:      "CREDIT_TRANSACTION",
		ID:        transaction.ID,
		UserID:    transaction.UserID,
		Amount:    transaction.Amount,
		TxType:    string(transaction.Type),
		Reason:    transaction.Reason,
		Metadata:  transaction.Metadata,
		CreatedAt: transaction.CreatedAt.Format(time.RFC3339),
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

func (r *creditTransactionRepositoryImpl) ListByUserID(ctx context.Context, userID string, limit int) ([]*domain.CreditTransaction, error) {
	if limit <= 0 {
		limit = 50 // default
	}

	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: "CREDITTX#"},
		},
		ScanIndexForward: aws.Bool(false), // Ordenar por mais recente primeiro
		Limit:            aws.Int32(int32(limit)),
	})

	if err != nil {
		return nil, err
	}

	transactions := make([]*domain.CreditTransaction, 0, len(result.Items))
	for _, item := range result.Items {
		var txItem CreditTransactionItem
		if err := attributevalue.UnmarshalMap(item, &txItem); err != nil {
			return nil, err
		}

		tx, err := r.itemToTransaction(&txItem)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *creditTransactionRepositoryImpl) itemToTransaction(item *CreditTransactionItem) (*domain.CreditTransaction, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &domain.CreditTransaction{
		ID:        item.ID,
		UserID:    item.UserID,
		Amount:    item.Amount,
		Type:      domain.CreditTransactionType(item.TxType),
		Reason:    item.Reason,
		Metadata:  item.Metadata,
		CreatedAt: createdAt,
	}, nil
}
