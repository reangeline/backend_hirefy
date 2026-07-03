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

type userRepositoryImpl struct {
	client *Client
}

func NewUserRepository(client *Client) outbound.UserRepository {
	return &userRepositoryImpl{client: client}
}

// UserItem representa o modelo no DynamoDB
type UserItem struct {
	PK               string `dynamodbav:"PK"`     // USER#<user_id>
	SK               string `dynamodbav:"SK"`     // METADATA
	GSI1PK           string `dynamodbav:"GSI1PK"` // EMAIL#<email>
	GSI1SK           string `dynamodbav:"GSI1SK"` // USER
	GSI2PK           string `dynamodbav:"GSI2PK"` // COGNITO#<cognito_id>
	GSI2SK           string `dynamodbav:"GSI2SK"` // USER
	Type             string `dynamodbav:"Type"`   // USER
	ID               string `dynamodbav:"ID"`
	Email            string `dynamodbav:"Email"`
	Name             string `dynamodbav:"Name"`
	Status           string `dynamodbav:"Status"`
	CognitoID        string `dynamodbav:"CognitoID"`
	StripeCustomerID string `dynamodbav:"StripeCustomerID,omitempty"`
	FCMToken         string `dynamodbav:"FCMToken,omitempty"`
	EmailVerified    bool   `dynamodbav:"EmailVerified"`
	TermsAcceptedAt  string `dynamodbav:"TermsAcceptedAt,omitempty"`
	TermsVersion     string `dynamodbav:"TermsVersion,omitempty"`
	CreatedAt        string `dynamodbav:"CreatedAt"`
	UpdatedAt        string `dynamodbav:"UpdatedAt"`
}

func (r *userRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	item := UserItem{
		PK:               fmt.Sprintf("USER#%s", user.ID),
		SK:               "METADATA",
		GSI1PK:           fmt.Sprintf("EMAIL#%s", user.Email),
		GSI1SK:           "USER",
		GSI2PK:           fmt.Sprintf("COGNITO#%s", user.CognitoID),
		GSI2SK:           "USER",
		Type:             "USER",
		ID:               user.ID,
		Email:            user.Email,
		Name:             user.Name,
		Status:           string(user.Status),
		CognitoID:        user.CognitoID,
		StripeCustomerID: user.StripeCustomerID,
		FCMToken:         user.FCMToken,
		EmailVerified:    user.EmailVerified,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        user.UpdatedAt.Format(time.RFC3339),
		TermsVersion:     user.TermsVersion,
	}

	if user.TermsAcceptedAt != nil {
		item.TermsAcceptedAt = user.TermsAcceptedAt.Format(time.RFC3339)
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

func (r *userRepositoryImpl) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, domain.ErrUserNotFound
	}

	var item UserItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}

	return r.itemToUser(&item)
}

func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :email AND GSI1SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: fmt.Sprintf("EMAIL#%s", email)},
			":sk":    &types.AttributeValueMemberS{Value: "USER"},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, domain.ErrUserNotFound
	}

	var item UserItem
	if err := attributevalue.UnmarshalMap(result.Items[0], &item); err != nil {
		return nil, err
	}

	return r.itemToUser(&item)
}

func (r *userRepositoryImpl) GetByCognitoID(ctx context.Context, cognitoID string) (*domain.User, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.tableName),
		IndexName:              aws.String("GSI2"),
		KeyConditionExpression: aws.String("GSI2PK = :pk AND GSI2SK = :sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("COGNITO#%s", cognitoID)},
			":sk": &types.AttributeValueMemberS{Value: "USER"},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, domain.ErrUserNotFound
	}

	var item UserItem
	if err := attributevalue.UnmarshalMap(result.Items[0], &item); err != nil {
		return nil, err
	}

	return r.itemToUser(&item)
}

func (r *userRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	updateExpr := "SET #name = :name, #status = :status, #stripeCustomerID = :stripeCustomerID, #emailVerified = :emailVerified, #fcmToken = :fcmToken, #updatedAt = :updatedAt"

	exprAttrNames := map[string]string{
		"#name":             "Name",
		"#status":           "Status",
		"#stripeCustomerID": "StripeCustomerID",
		"#emailVerified":    "EmailVerified",
		"#fcmToken":         "FCMToken",
		"#updatedAt":        "UpdatedAt",
	}

	exprAttrValues := map[string]types.AttributeValue{
		":name":             &types.AttributeValueMemberS{Value: user.Name},
		":status":           &types.AttributeValueMemberS{Value: string(user.Status)},
		":stripeCustomerID": &types.AttributeValueMemberS{Value: user.StripeCustomerID},
		":emailVerified":    &types.AttributeValueMemberBOOL{Value: user.EmailVerified},
		":fcmToken":         &types.AttributeValueMemberS{Value: user.FCMToken},
		":updatedAt":        &types.AttributeValueMemberS{Value: user.UpdatedAt.Format(time.RFC3339)},
	}

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", user.ID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	return err
}

func (r *userRepositoryImpl) Delete(ctx context.Context, userID string) error {
	_, err := r.client.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})

	return err
}

// PurgeAllUserItems repeatedly queries the user's partition until it is empty
// and deletes items in batches of 25 (the DynamoDB BatchWriteItem limit).
// Re-querying from the start avoids skipping items while deleting the same
// partition that is being paginated.
func (r *userRepositoryImpl) PurgeAllUserItems(ctx context.Context, userID string) error {
	pk := fmt.Sprintf("USER#%s", userID)

	for {
		queryOut, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(r.client.tableName),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: pk},
			},
			ProjectionExpression: aws.String("PK, SK"),
			ConsistentRead:       aws.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("PurgeAllUserItems query failed: %w", err)
		}

		if len(queryOut.Items) == 0 {
			break
		}

		// Batch delete up to 25 items per BatchWriteItem call.
		for i := 0; i < len(queryOut.Items); i += 25 {
			end := i + 25
			if end > len(queryOut.Items) {
				end = len(queryOut.Items)
			}
			chunk := queryOut.Items[i:end]

			var writeReqs []types.WriteRequest
			for _, item := range chunk {
				writeReqs = append(writeReqs, types.WriteRequest{
					DeleteRequest: &types.DeleteRequest{
						Key: map[string]types.AttributeValue{
							"PK": item["PK"],
							"SK": item["SK"],
						},
					},
				})
			}

			out, err := r.client.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{
					r.client.tableName: writeReqs,
				},
			})
			if err != nil {
				return fmt.Errorf("PurgeAllUserItems batch delete failed: %w", err)
			}

			// Retry any unprocessed items (e.g. due to DynamoDB throttling).
			for len(out.UnprocessedItems) > 0 {
				out, err = r.client.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
					RequestItems: out.UnprocessedItems,
				})
				if err != nil {
					return fmt.Errorf("PurgeAllUserItems retry failed: %w", err)
				}
			}
		}
	}

	return nil
}

func (r *userRepositoryImpl) UpdateFCMToken(ctx context.Context, userID, token string) error {
	updateExpr := "SET #fcmToken = :fcmToken, #updatedAt = :updatedAt"

	exprAttrNames := map[string]string{
		"#fcmToken":  "FCMToken",
		"#updatedAt": "UpdatedAt",
	}

	exprAttrValues := map[string]types.AttributeValue{
		":fcmToken":  &types.AttributeValueMemberS{Value: token},
		":updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}

	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
	})

	return err
}

func (r *userRepositoryImpl) itemToUser(item *UserItem) (*domain.User, error) {
	createdAt, err := time.Parse(time.RFC3339, item.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := time.Parse(time.RFC3339, item.UpdatedAt)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:               item.ID,
		Email:            item.Email,
		Name:             item.Name,
		Status:           domain.UserStatus(item.Status),
		CognitoID:        item.CognitoID,
		StripeCustomerID: item.StripeCustomerID,
		FCMToken:         item.FCMToken,
		EmailVerified:    item.EmailVerified,
		TermsVersion:     item.TermsVersion,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	if item.TermsAcceptedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.TermsAcceptedAt); err == nil {
			user.TermsAcceptedAt = &t
		}
	}

	return user, nil
}

// Design de tabela única (Single Table Design):
// PK                    SK              GSI1PK              GSI1SK    GSI2PK              GSI2SK         Type
// USER#<user_id>        METADATA        EMAIL#<email>       USER      COGNITO#<cognito>   USER           USER
// USER#<user_id>        SUB#<sub_id>    -                   -         STRIPESUB#<sub_id>  SUBSCRIPTION   SUBSCRIPTION
// USER#<user_id>        RES#<res_id>    -                   -         -                   -              RESUME
// USER#<user_id>        OPT#<opt_id>
