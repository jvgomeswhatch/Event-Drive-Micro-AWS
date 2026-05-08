package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
	"github.com/platform/payment-service/internal/domain"
	"github.com/platform/payment-service/internal/logger"
)

type Processor struct {
	db       *dynamodb.Client
	sns      *sns.Client
	idempot  *idempotencyClient
	payments string
	timeline string
	topicARN string
}

type idempotencyClient struct {
	db    *dynamodb.Client
	table string
}

func New(db *dynamodb.Client, snsClient *sns.Client) *Processor {
	idem := &idempotencyClient{
		db:    db,
		table: getenv("IDEMPOTENCY_TABLE", "idempotency-dev"),
	}
	return &Processor{
		db:       db,
		sns:      snsClient,
		idempot:  idem,
		payments: getenv("PAYMENTS_TABLE", "payments-dev"),
		timeline: getenv("EVENT_TIMELINE_TABLE", "event-timeline-dev"),
		topicARN: os.Getenv("PAYMENT_EVENTS_TOPIC_ARN"),
	}
}

func (p *Processor) Process(ctx context.Context, event domain.OrderCreatedEvent) error {
	correlationID := event.Meta.CorrelationID
	idemKey := "payment-service#" + event.OrderID

	claimed, err := p.idempot.claim(ctx, idemKey)
	if err != nil {
		return fmt.Errorf("idempotency claim: %w", err)
	}
	if !claimed {
		logger.Info("Duplicate payment event — skipping", logger.Fields{"correlationId": correlationID, "orderId": event.OrderID})
		return nil
	}

	paymentID := uuid.New().String()
	now := time.Now().UTC()

	// Calculate total (stub: $10 per unit)
	var total float64
	for _, item := range event.Items {
		total += float64(item.Quantity) * 10.0
	}

	// Determine outcome: manual flag OR 5% random failure
	failed := event.SimulateFailure || rand.Float64() < 0.05
	status := "succeeded"
	failureReason := ""
	if failed {
		status = "failed"
		if event.SimulateFailure {
			failureReason = "Manually simulated failure"
		} else {
			failureReason = "Insufficient funds"
		}
	}

	payment := domain.Payment{
		PaymentID:     paymentID,
		OrderID:       event.OrderID,
		CustomerID:    event.CustomerID,
		TotalAmount:   total,
		Status:        status,
		FailureReason: failureReason,
		CorrelationID: correlationID,
		CreatedAt:     now.Format(time.RFC3339),
	}

	// Persist payment
	paymentItem, err := attributevalue.MarshalMap(payment)
	if err != nil {
		return fmt.Errorf("marshal payment: %w", err)
	}
	if _, err := p.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(p.payments),
		Item:      paymentItem,
	}); err != nil {
		return fmt.Errorf("persist payment: %w", err)
	}

	// Append to event timeline
	timelineItem, _ := attributevalue.MarshalMap(map[string]any{
		"orderId":       event.OrderID,
		"eventId":       "payment#" + paymentID,
		"service":       "payment-service",
		"eventType":     "payment." + status,
		"payload":       map[string]any{"paymentId": paymentID, "totalAmount": total, "status": status, "failureReason": failureReason},
		"correlationId": correlationID,
		"timestamp":     now.Format(time.RFC3339),
		"expiresAt":     now.Unix() + 7*86400,
	})
	_, _ = p.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(p.timeline), Item: timelineItem})

	// Publish SNS event
	if p.topicARN != "" {
		resultEvent := domain.PaymentResultEvent{
			EventType:     "payment." + status,
			Version:       "1",
			PaymentID:     paymentID,
			OrderID:       event.OrderID,
			CustomerID:    event.CustomerID,
			Items:         event.Items,
			TotalAmount:   total,
			Status:        status,
			FailureReason: failureReason,
			Meta: domain.EventMeta{
				CorrelationID: correlationID,
				PublishedAt:   now.Format(time.RFC3339),
				Publisher:     "payment-service",
			},
		}
		msgBytes, _ := json.Marshal(resultEvent)
		if _, snsErr := p.sns.Publish(ctx, &sns.PublishInput{
			TopicArn: aws.String(p.topicARN),
			Message:  aws.String(string(msgBytes)),
			MessageAttributes: map[string]snstypes.MessageAttributeValue{
				"eventType":     {DataType: aws.String("String"), StringValue: aws.String("payment." + status)},
				"correlationId": {DataType: aws.String("String"), StringValue: aws.String(correlationID)},
			},
		}); snsErr != nil {
			// Payment already persisted — resolve idempotency so we don't recharge on retry,
			// but propagate the error so the caller can decide (e.g. alert, DLQ).
			_ = p.idempot.resolve(ctx, idemKey, map[string]string{"paymentId": paymentID, "status": status})
			return fmt.Errorf("publish payment event to SNS: %w", snsErr)
		}
	}

	_ = p.idempot.resolve(ctx, idemKey, map[string]string{"paymentId": paymentID, "status": status})
	logger.Info("Payment processed", logger.Fields{
		"correlationId": correlationID,
		"orderId":       event.OrderID,
		"paymentId":     paymentID,
		"status":        status,
		"totalAmount":   fmt.Sprintf("%.2f", total),
	})

	if failed {
		return fmt.Errorf("payment failed: %s", failureReason)
	}
	return nil
}

func (c *idempotencyClient) claim(ctx context.Context, key string) (bool, error) {
	item, _ := attributevalue.MarshalMap(map[string]any{
		"idempotencyKey": key,
		"status":         "processing",
		"createdAt":      time.Now().UTC().Format(time.RFC3339),
		"expiresAt":      time.Now().Unix() + 86400,
	})
	_, err := c.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(c.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(idempotencyKey)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			// Chave já existe → evento duplicado legítimo
			return false, nil
		}
		// Qualquer outro erro (rede, timeout, permissão) deve propagar
		return false, fmt.Errorf("idempotency claim failed: %w", err)
	}
	return true, nil
}

func (c *idempotencyClient) resolve(ctx context.Context, key string, result map[string]string) error {
	item, _ := attributevalue.MarshalMap(map[string]any{
		"idempotencyKey": key,
		"status":         "completed",
		"result":         result,
		"expiresAt":      time.Now().Unix() + 86400,
	})
	_, err := c.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(c.table), Item: item})
	return err
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
