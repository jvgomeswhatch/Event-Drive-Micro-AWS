package idempotency

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const ttlSeconds = 86400

var ErrAlreadyProcessed = errors.New("idempotency: key already processed")

type Record struct {
	IdempotencyKey string `dynamodbav:"idempotencyKey"`
	Status         string `dynamodbav:"status"`
	Result         any    `dynamodbav:"result,omitempty"`
	CreatedAt      string `dynamodbav:"createdAt"`
	ExpiresAt      int64  `dynamodbav:"expiresAt"`
}

type Client struct {
	db    *dynamodb.Client
	table string
}

func New(db *dynamodb.Client) *Client {
	table := os.Getenv("IDEMPOTENCY_TABLE")
	if table == "" {
		table = "idempotency-dev"
	}
	return &Client{db: db, table: table}
}

// ClaimKey attempts to atomically claim the key.
// Returns (true, nil) on first claim.
// Returns (false, nil) if already completed — caller should skip processing.
// Returns (false, err) on unexpected error.
func (c *Client) ClaimKey(ctx context.Context, key string) (claimed bool, err error) {
	item, err := attributevalue.MarshalMap(Record{
		IdempotencyKey: key,
		Status:         "processing",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:      time.Now().Unix() + ttlSeconds,
	})
	if err != nil {
		return false, err
	}

	_, err = c.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(c.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(idempotencyKey)"),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ResolveKey marks the key as completed with the given result.
func (c *Client) ResolveKey(ctx context.Context, key string, result map[string]string) error {
	item, err := attributevalue.MarshalMap(Record{
		IdempotencyKey: key,
		Status:         "completed",
		Result:         result,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:      time.Now().Unix() + ttlSeconds,
	})
	if err != nil {
		return err
	}
	_, err = c.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(c.table),
		Item:      item,
	})
	return err
}
