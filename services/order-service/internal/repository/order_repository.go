package repository

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/platform/order-service/internal/domain"
)

type OrderRepository struct {
	db    *dynamodb.Client
	table string
}

func NewOrderRepository(db *dynamodb.Client) *OrderRepository {
	table := os.Getenv("ORDERS_TABLE")
	if table == "" {
		table = "orders-dev"
	}
	return &OrderRepository{db: db, table: table}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	item, err := attributevalue.MarshalMap(order)
	if err != nil {
		return err
	}
	_, err = r.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(orderId)"),
	})
	return err
}

func (r *OrderRepository) GetByID(ctx context.Context, orderID string) (*domain.Order, error) {
	out, err := r.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"orderId": &types.AttributeValueMemberS{Value: orderID},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var order domain.Order
	if err := attributevalue.UnmarshalMap(out.Item, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID string, limit int32) ([]domain.Order, error) {
	out, err := r.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.table),
		IndexName:              aws.String("customerId-createdAt-index"),
		KeyConditionExpression: aws.String("customerId = :cid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cid": &types.AttributeValueMemberS{Value: customerID},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}
	var orders []domain.Order
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}
