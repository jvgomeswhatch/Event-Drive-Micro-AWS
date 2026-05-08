package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/platform/order-service/internal/eventschema"
	"github.com/platform/order-service/internal/logger"
	"github.com/platform/order-service/internal/tracing"
)

type SNSPublisher struct {
	client    *sns.Client
	topicARN  string
	topicName string
}

func NewSNSPublisher(client *sns.Client) *SNSPublisher {
	topicARN := os.Getenv("ORDER_EVENTS_TOPIC_ARN")
	topicName := os.Getenv("ORDER_EVENTS_TOPIC_NAME")
	if topicName == "" {
		topicName = "order-events-dev"
	}
	return &SNSPublisher{client: client, topicARN: topicARN, topicName: topicName}
}

func (p *SNSPublisher) resolveARN(ctx context.Context) error {
	if p.topicARN != "" {
		return nil
	}
	out, err := p.client.CreateTopic(ctx, &sns.CreateTopicInput{
		Name: aws.String(p.topicName),
	})
	if err != nil {
		return fmt.Errorf("sns publisher: resolve topic ARN: %w", err)
	}
	p.topicARN = *out.TopicArn
	return nil
}

func (p *SNSPublisher) Publish(ctx context.Context, event any, correlationID, eventType string) error {
	ctx, span := tracing.Start(ctx, "sns.publish."+eventType)
	defer span.Finish(nil)
	span.Tag("topic", p.topicName).Tag("eventType", eventType)

	if err := p.resolveARN(ctx); err != nil {
		return err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("sns publisher: marshal event: %w", err)
	}

	if err := eventschema.Validate(eventType, eventschema.V1, body); err != nil {
		return fmt.Errorf("sns publisher: schema violation: %w", err)
	}

	attrs := map[string]snstypes.MessageAttributeValue{
		"correlationId": {DataType: aws.String("String"), StringValue: aws.String(correlationID)},
		"eventType":     {DataType: aws.String("String"), StringValue: aws.String(eventType)},
		"spanId":        {DataType: aws.String("String"), StringValue: aws.String(span.SpanID)},
	}

	out, err := p.client.Publish(ctx, &sns.PublishInput{
		TopicArn:          aws.String(p.topicARN),
		Message:           aws.String(string(body)),
		MessageAttributes: attrs,
	})
	if err != nil {
		return fmt.Errorf("sns publisher: publish message: %w", err)
	}

	logger.Info("Event published to SNS", logger.Fields{
		"correlationId": correlationID,
		"eventType":     eventType,
		"spanId":        span.SpanID,
		"messageId":     *out.MessageId,
		"topic":         p.topicName,
	})
	return nil
}
