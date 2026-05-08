package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
	"github.com/platform/inventory-service/internal/domain"
	"github.com/platform/inventory-service/internal/logger"
)

type Processor struct {
	db       *dynamodb.Client
	sns      *sns.Client
	inventory string
	timeline  string
	topicARN  string
	idemTable string
}

func New(db *dynamodb.Client, snsClient *sns.Client) *Processor {
	return &Processor{
		db:        db,
		sns:       snsClient,
		inventory: getenv("INVENTORY_TABLE", "inventory-dev"),
		timeline:  getenv("EVENT_TIMELINE_TABLE", "event-timeline-dev"),
		topicARN:  os.Getenv("INVENTORY_EVENTS_TOPIC_ARN"),
		idemTable: getenv("IDEMPOTENCY_TABLE", "idempotency-dev"),
	}
}

func (p *Processor) Process(ctx context.Context, event domain.PaymentSucceededEvent) error {
	correlationID := event.Meta.CorrelationID
	idemKey := "inventory-service#" + event.OrderID

	// Idempotency
	idemItem, _ := attributevalue.MarshalMap(map[string]any{
		"idempotencyKey": idemKey, "status": "processing",
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"expiresAt": time.Now().Unix() + 86400,
	})
	_, err := p.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(p.idemTable),
		Item:                idemItem,
		ConditionExpression: aws.String("attribute_not_exists(idempotencyKey)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			logger.Info("Duplicate inventory event — skipping", logger.Fields{"correlationId": correlationID, "orderId": event.OrderID})
			return nil
		}
		return fmt.Errorf("idempotency claim failed: %w", err)
	}

	reservationID := uuid.New().String()
	now := time.Now().UTC()
	status := "reserved"
	failureReason := ""
	reserved := []domain.OrderItem{}

	// Reserve each item atomically with conditional expression
	for _, item := range event.Items {
		qty := item.Quantity
		_, err := p.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(p.inventory),
			Key: map[string]types.AttributeValue{
				"productId": &types.AttributeValueMemberS{Value: item.ProductID},
			},
			UpdateExpression:    aws.String("SET reserved = reserved + :qty, quantity = quantity - :qty"),
			ConditionExpression: aws.String("quantity >= :qty AND attribute_exists(productId)"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":qty": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", qty)},
			},
		})
		if err != nil {
			// Conditional check failed → insufficient stock or product not found
			failureReason = fmt.Sprintf("Insufficient stock or product not found: %s (requested %d)", item.ProductID, qty)
			status = "failed"
			// Rollback already-reserved items
			p.rollback(ctx, reserved, correlationID)
			break
		}
		reserved = append(reserved, item)
	}

	// Append to event timeline
	tlItem, _ := attributevalue.MarshalMap(map[string]any{
		"orderId": event.OrderID, "eventId": "inventory#" + reservationID,
		"service": "inventory-service", "eventType": "inventory." + status,
		"payload":       map[string]any{"reservationId": reservationID, "items": event.Items, "status": status, "failureReason": failureReason},
		"correlationId": correlationID, "timestamp": now.Format(time.RFC3339),
		"expiresAt": now.Unix() + 7*86400,
	})
	_, _ = p.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(p.timeline), Item: tlItem})

	// Publish to SNS
	if p.topicARN != "" {
		resultEvent := domain.InventoryResultEvent{
			EventType: "inventory." + status, Version: "1",
			ReservationID: reservationID, OrderID: event.OrderID,
			CustomerID: event.CustomerID, Items: event.Items,
			Status: status, FailureReason: failureReason,
			Meta: domain.EventMeta{CorrelationID: correlationID, PublishedAt: now.Format(time.RFC3339), Publisher: "inventory-service"},
		}
		msgBytes, _ := json.Marshal(resultEvent)
		if _, snsErr := p.sns.Publish(ctx, &sns.PublishInput{
			TopicArn: aws.String(p.topicARN),
			Message:  aws.String(string(msgBytes)),
			MessageAttributes: map[string]snstypes.MessageAttributeValue{
				"eventType":     {DataType: aws.String("String"), StringValue: aws.String("inventory." + status)},
				"correlationId": {DataType: aws.String("String"), StringValue: aws.String(correlationID)},
			},
		}); snsErr != nil {
			logger.Error("Failed to publish inventory SNS event", logger.Fields{
				"correlationId": correlationID, "orderId": event.OrderID, "error": snsErr.Error(),
			})
			return fmt.Errorf("publish inventory event to SNS: %w", snsErr)
		}
	}

	// Mark idempotency complete
	idemDone, _ := attributevalue.MarshalMap(map[string]any{
		"idempotencyKey": idemKey, "status": "completed",
		"result": map[string]string{"reservationId": reservationID, "status": status},
		"expiresAt": time.Now().Unix() + 86400,
	})
	_, _ = p.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(p.idemTable), Item: idemDone})

	logger.Info("Inventory processed", logger.Fields{
		"correlationId": correlationID, "orderId": event.OrderID,
		"reservationId": reservationID, "status": status,
	})
	return nil
}

func (p *Processor) rollback(ctx context.Context, items []domain.OrderItem, correlationID string) {
	for _, item := range items {
		_, err := p.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(p.inventory),
			Key: map[string]types.AttributeValue{
				"productId": &types.AttributeValueMemberS{Value: item.ProductID},
			},
			UpdateExpression: aws.String("SET reserved = reserved - :qty, quantity = quantity + :qty"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":qty": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", item.Quantity)},
			},
		})
		if err != nil {
			logger.Error("Rollback failed — manual intervention required", logger.Fields{
				"correlationId": correlationID, "productId": item.ProductID, "error": err.Error(),
			})
		}
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
