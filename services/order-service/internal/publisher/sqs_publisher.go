package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/platform/order-service/internal/eventschema"
	"github.com/platform/order-service/internal/logger"
	"github.com/platform/order-service/internal/tracing"
)

type SQSPublisher struct {
	client    *sqs.Client
	queueName string
	queueURL  string
}

func NewSQSPublisher(client *sqs.Client) *SQSPublisher {
	name := os.Getenv("ORDER_QUEUE_NAME")
	if name == "" {
		name = "order-queue-dev"
	}
	return &SQSPublisher{client: client, queueName: name}
}

func (p *SQSPublisher) resolveURL(ctx context.Context) error {
	if p.queueURL != "" {
		return nil
	}
	out, err := p.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(p.queueName),
	})
	if err != nil {
		return fmt.Errorf("publisher: resolve queue URL: %w", err)
	}
	p.queueURL = *out.QueueUrl
	return nil
}

func (p *SQSPublisher) Publish(ctx context.Context, event any, correlationID, eventType string) error {
	ctx, span := tracing.Start(ctx, "sqs.publish."+eventType)
	defer span.Finish(nil)
	span.Tag("queue", p.queueName).Tag("eventType", eventType)

	if err := p.resolveURL(ctx); err != nil {
		return err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("publisher: marshal event: %w", err)
	}

	// Validate against schema before sending — fail fast at publish time
	if err := eventschema.Validate(eventType, eventschema.V1, body); err != nil {
		return fmt.Errorf("publisher: schema violation: %w", err)
	}

	// Propagate trace context as SQS message attributes
	attrs := map[string]types.MessageAttributeValue{
		"correlationId": {DataType: aws.String("String"), StringValue: aws.String(correlationID)},
		"eventType":     {DataType: aws.String("String"), StringValue: aws.String(eventType)},
		"spanId":        {DataType: aws.String("String"), StringValue: aws.String(span.SpanID)},
	}

	out, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(p.queueURL),
		MessageBody:       aws.String(string(body)),
		MessageAttributes: attrs,
	})
	if err != nil {
		return fmt.Errorf("publisher: send message: %w", err)
	}

	logger.Info("Event published to SQS", logger.Fields{
		"correlationId": correlationID,
		"eventType":     eventType,
		"spanId":        span.SpanID,
		"messageId":     *out.MessageId,
		"queue":         p.queueName,
	})
	return nil
}
