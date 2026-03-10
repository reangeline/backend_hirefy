package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Client struct {
	db        *dynamodb.Client
	tableName string
}

func NewClient(cfg aws.Config, tableName string) *Client {
	return &Client{
		db:        dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}
