package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/platform/inventory-service/internal/awsclient"
	"github.com/platform/inventory-service/internal/domain"
	"github.com/platform/inventory-service/internal/eventschema"
	"github.com/platform/inventory-service/internal/logger"
	"github.com/platform/inventory-service/internal/processor"
	"github.com/platform/inventory-service/internal/tracing"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqsClient := awsclient.NewSQS(ctx)
	dynamo := awsclient.NewDynamo(ctx)
	snsClient := awsclient.NewSNS(ctx)
	proc := processor.New(dynamo, snsClient)

	go startHealthServer(getenv("INVENTORY_SERVICE_PORT", "3003"))

	queueName := getenv("INVENTORY_QUEUE_NAME", "inventory-queue-dev")
	queueURL := resolveQueueURL(ctx, sqsClient, queueName)

	logger.Info("inventory-service consumer started", logger.Fields{"queue": queueName})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; cancel() }()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		out, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl: aws.String(queueURL), MaxNumberOfMessages: 10, WaitTimeSeconds: 20,
			MessageAttributeNames: []string{"All"},
			AttributeNames:        []sqstypes.QueueAttributeName{"ApproximateReceiveCount"},
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("SQS receive error", logger.Fields{"error": err.Error()})
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range out.Messages {
			go handleMessage(ctx, sqsClient, queueURL, msg, proc)
		}
	}
}

func handleMessage(ctx context.Context, sqsClient *sqs.Client, queueURL string, msg sqstypes.Message, proc *processor.Processor) {
	receiveCount := 1
	if v, ok := msg.Attributes["ApproximateReceiveCount"]; ok {
		receiveCount, _ = strconv.Atoi(v)
	}

	raw := []byte(*msg.Body)

	eventType, version := eventschema.ExtractMeta(raw)
	if eventType == "payment.succeeded" {
		if err := eventschema.Validate(eventType, eventschema.Version(version), raw); err != nil {
			logger.Error("Schema validation failed — routing to DLQ", logger.Fields{
				"messageId": *msg.MessageId, "eventType": eventType, "error": err.Error(),
			})
			changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
			return
		}
	}

	var event domain.PaymentSucceededEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		logger.Error("Unparseable message", logger.Fields{"messageId": *msg.MessageId})
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
		return
	}

	// Only handle payment.succeeded
	if event.EventType != "payment.succeeded" {
		deleteMessage(ctx, sqsClient, queueURL, *msg.ReceiptHandle)
		return
	}

	parentSpanID := ""
	if attr, ok := msg.MessageAttributes["spanId"]; ok && attr.StringValue != nil {
		parentSpanID = *attr.StringValue
	}
	ctx, span := tracing.StartWithTrace(ctx, event.Meta.CorrelationID, parentSpanID, "inventory.process")
	span.Tag("orderId", event.OrderID)

	err := proc.Process(ctx, event)
	if err == nil {
		span.Finish(nil)
		deleteMessage(ctx, sqsClient, queueURL, *msg.ReceiptHandle)
		return
	}
	span.Finish(err)

	logger.Warn("Inventory processing failed", logger.Fields{
		"correlationId": event.Meta.CorrelationID, "messageId": *msg.MessageId,
		"receiveCount": receiveCount, "error": err.Error(),
	})
	if receiveCount >= 3 {
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
	} else {
		backoff := int32(30 * receiveCount)
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, backoff)
	}
}

func resolveQueueURL(ctx context.Context, client *sqs.Client, name string) string {
	for {
		out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
		if err == nil {
			return *out.QueueUrl
		}
		logger.Warn("Queue not ready", logger.Fields{"queue": name, "error": err.Error()})
		time.Sleep(3 * time.Second)
	}
}

func deleteMessage(ctx context.Context, c *sqs.Client, url, receipt string) {
	_, _ = c.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: aws.String(url), ReceiptHandle: aws.String(receipt)})
}

func changeVisibility(ctx context.Context, c *sqs.Client, url, receipt string, t int32) {
	_, _ = c.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{QueueUrl: aws.String(url), ReceiptHandle: aws.String(receipt), VisibilityTimeout: t})
}

func startHealthServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","service":"inventory-service"}`)
	})
	_ = http.ListenAndServe(":"+port, mux)
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
