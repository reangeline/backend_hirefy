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

type subscriptionRepositoryImpl struct {
	client *Client
}

func NewSubscriptionRepository(client *Client) outbound.SubscriptionRepository {
	return &subscriptionRepositoryImpl{client: client}
}

type SubscriptionItem struct {
	PK               string  `dynamodbav:"PK"`     // USER#<user_id>
	SK               string  `dynamodbav:"SK"`     // SUB#<sub_id>
	GSI2PK           string  `dynamodbav:"GSI2PK"` // STRIPESUB#<stripe_sub_id>
	GSI2SK           string  `dynamodbav:"GSI2SK"` // SUBSCRIPTION
	Type             string  `dynamodbav:"Type"`   // SUBSCRIPTION
	ID               string  `dynamodbav:"ID"`
	UserID           string  `dynamodbav:"UserID"`
	Plan             string  `dynamodbav:"Plan"`
	Status           string  `dynamodbav:"Status"`
	StripeCustomerID string  `dynamodbav:"StripeCustomerID"`
	StripeSubID      string  `dynamodbav:"StripeSubID"`
	CurrentPeriodEnd string  `dynamodbav:"CurrentPeriodEnd"`
	CreatedAt        string  `dynamodbav:"CreatedAt"`
	UpdatedAt        string  `dynamodbav:"UpdatedAt"`
	CanceledAt       *string `dynamodbav:"CanceledAt,omitempty"`
}

func (r *subscriptionRepositoryImpl) Create(ctx context.Context, subscription *domain.Subscription) error {
	var canceledAt *string
	if subscription.CanceledAt != nil {
		t := subscription.CanceledAt.Format(time.RFC3339)
		canceledAt = &t
	}

	item := SubscriptionItem{
		PK:               fmt.Sprintf("USER#%s", subscription.UserID),
		SK:               fmt.Sprintf("SUB#%s", subscription.ID),
		GSI2PK:           fmt.Sprintf("STRIPESUB#%s", subscription.StripeSubID),
		GSI2SK:           "SUBSCRIPTION",
		Type:             "SUBSCRIPTION",
		ID:               subscription.ID,
		UserID:           subscription.UserID,
		Plan:             string(subscription.Plan),
		Status:           string(subscription.Status),
		StripeCustomerID: subscription.StripeCustomerID,
		StripeSubID:      subscription.StripeSubID,
		CurrentPeriodEnd: subscription.CurrentPeriodEnd.Format(time.RFC3339),
		CreatedAt:        subscription.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        subscription.UpdatedAt.Format(time.RFC3339),
		CanceledAt:       canceledAt,
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

func (r *subscriptionRepositoryImpl) GetByID(ctx context.Context, subscriptionID string) (*domain.Subscription, error) {
	// Para buscar por ID sem saber o UserID, precisamos fazer um scan ou ter um GSI
	// Por simplicidade, vou assumir que sempre buscaremos por UserID
	return nil, fmt.Errorf("not implemented: use GetByUserID instead")
}

func (r *subscriptionRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			":sk": &types.AttributeValueMemberS{Value: "SUB#"},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, domain.ErrSubscriptionNotFound
	}

	var item SubscriptionItem
	if err := attributevalue.UnmarshalMap(result.Items[0], &item); err != nil {
		return nil, err
	}

	return r.itemToSubscription(&item)
}

func (r *subscriptionRepositoryImpl) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk AND GSI2SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("STRIPESUB#%s", stripeSubID)},
			":sk": &types.AttributeValueMemberS{Value: "SUBSCRIPTION"},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, domain.ErrSubscriptionNotFound
	}

	var item SubscriptionItem
	if err := attributevalue.UnmarshalMap(result.Items[0], &item); err != nil {
		return nil, err
	}

	return r.itemToSubscription(&item)
}

func (r *subscriptionRepositoryImpl) Update(ctx context.Context, subscription *domain.Subscription) error {
	var canceledAt *string
	if subscription.CanceledAt != nil {
		t := subscription.CanceledAt.Format(time.RFC3339)
		canceledAt = &t
	}

	updateExpr := "SET #plan = :plan, #status = :status, #currentPeriodEnd = :currentPeriodEnd, #updatedAt = :updatedAt"
	exprAttrNames := map[string]string{
		"#plan":             "Plan",
		"#status":           "Status",
		"#currentPeriodEnd": "CurrentPeriodEnd",
		"#updatedAt":        "UpdatedAt",
	}
	exprAttrValues := map[string]types.AttributeValue{
		":plan":             &types.AttributeValueMemberS{Value: string(subscription.Plan)},
		":status":           &types.AttributeValueMemberS{Value: string(subscription.Status)},
		":currentPeriodEnd": &types.AttributeValueMemberS{Value: subscription.CurrentPeriodEnd.Format(time.RFC3339)},
		":updatedAt":        &types.AttributeValueMemberS{Value: subscription.UpdatedAt.Format(time.RFC3339)},
	}

	if canceledAt != nil {
		updateExpr += ", #canceledAt = :canceledAt"
		exprAttrNames["#canceledAt"] = "CanceledAt"
		exprAttrValues[":canceledAt"] = &types.AttributeValueMemberS{Value: *canceledAt}
	}

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", subscription.UserID)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("SUB#%s", subscription.ID)},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	return err
}

func (r *subscriptionRepositoryImpl) itemToSubscription(item *SubscriptionItem) (*domain.Subscription, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	currentPeriodEnd, err := time.Parse(time.RFC3339, item.CurrentPeriodEnd)
	if err != nil {
		return nil, err
	}

	var canceledAt *time.Time
	if item.CanceledAt != nil {
		t, err := time.Parse(time.RFC3339, *item.CanceledAt)
		if err != nil {
			return nil, err
		}
		canceledAt = &t
	}

	return &domain.Subscription{
		ID:               item.ID,
		UserID:           item.UserID,
		Plan:             domain.SubscriptionPlan(item.Plan),
		Status:           domain.SubscriptionStatus(item.Status),
		StripeCustomerID: item.StripeCustomerID,
		StripeSubID:      item.StripeSubID,
		CurrentPeriodEnd: currentPeriodEnd,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		CanceledAt:       canceledAt,
	}, nil
}
