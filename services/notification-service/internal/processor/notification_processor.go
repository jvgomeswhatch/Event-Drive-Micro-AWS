package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/platform/notification-service/internal/domain"
	"github.com/platform/notification-service/internal/logger"
)

type Processor struct {
	db        *dynamodb.Client
	orders    string
	timeline  string
	idemTable string
}

func New(db *dynamodb.Client) *Processor {
	return &Processor{
		db:        db,
		orders:    getenv("ORDERS_TABLE", "orders-dev"),
		timeline:  getenv("EVENT_TIMELINE_TABLE", "event-timeline-dev"),
		idemTable: getenv("IDEMPOTENCY_TABLE", "idempotency-dev"),
	}
}

func (p *Processor) Process(ctx context.Context, event domain.IncomingEvent) error {
	correlationID := event.Meta.CorrelationID
	idemKey := fmt.Sprintf("notification-service#%s#%s", event.OrderID, event.EventType)

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
			logger.Info("Duplicate notification event — skipping", logger.Fields{"correlationId": correlationID, "orderId": event.OrderID, "eventType": event.EventType})
			return nil
		}
		return fmt.Errorf("idempotency claim failed: %w", err)
	}

	newStatus, ok := domain.StatusMap[event.EventType]
	if !ok {
		return nil
	}

	now := time.Now().UTC()

	// Update order status
	_, err = p.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(p.orders),
		Key: map[string]types.AttributeValue{
			"orderId": &types.AttributeValueMemberS{Value: event.OrderID},
		},
		UpdateExpression:    aws.String("SET #st = :status, updatedAt = :updatedAt, lastEvent = :lastEvent"),
		ExpressionAttributeNames: map[string]string{"#st": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: newStatus},
			":updatedAt": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":lastEvent": &types.AttributeValueMemberS{Value: event.EventType},
		},
	})
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	notificationID := uuid.New().String()

	// Append to timeline
	tlItem, _ := attributevalue.MarshalMap(map[string]any{
		"orderId": event.OrderID, "eventId": "notification#" + notificationID,
		"service": "notification-service", "eventType": "notification.sent",
		"payload": map[string]any{
			"notificationId": notificationID,
			"triggeredBy":    event.EventType,
			"newStatus":      newStatus,
			"message":        buildMessage(event),
		},
		"correlationId": correlationID,
		"timestamp":     now.Format(time.RFC3339),
		"expiresAt":     now.Unix() + 7*86400,
	})
	_, _ = p.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(p.timeline), Item: tlItem})

	// Structured log as simulated notification dispatch
	logger.Info("Notification dispatched", logger.Fields{
		"correlationId":  correlationID,
		"orderId":        event.OrderID,
		"notificationId": notificationID,
		"channel":        "email+websocket",
		"newStatus":      newStatus,
		"eventType":      event.EventType,
		"message":        buildMessage(event),
	})

	// Mark complete
	idemDone, _ := attributevalue.MarshalMap(map[string]any{
		"idempotencyKey": idemKey, "status": "completed",
		"result": map[string]string{"notificationId": notificationID, "newStatus": newStatus},
		"expiresAt": time.Now().Unix() + 86400,
	})
	_, _ = p.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(p.idemTable), Item: idemDone})

	return nil
}

func buildMessage(event domain.IncomingEvent) string {
	switch event.EventType {
	case "payment.succeeded":
		return fmt.Sprintf("Your payment of $%.2f was successful.", event.TotalAmount)
	case "payment.failed":
		return fmt.Sprintf("Payment failed: %s. Please retry.", event.FailureReason)
	case "inventory.reserved":
		return "Your items have been reserved and your order is confirmed."
	case "inventory.failed":
		return fmt.Sprintf("Order could not be fulfilled: %s.", event.FailureReason)
	default:
		return fmt.Sprintf("Order update: %s", event.EventType)
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
