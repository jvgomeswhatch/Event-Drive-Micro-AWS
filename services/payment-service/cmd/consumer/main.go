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
	"github.com/platform/payment-service/internal/awsclient"
	"github.com/platform/payment-service/internal/domain"
	"github.com/platform/payment-service/internal/eventschema"
	"github.com/platform/payment-service/internal/logger"
	"github.com/platform/payment-service/internal/processor"
	"github.com/platform/payment-service/internal/tracing"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqsClient := awsclient.NewSQS(ctx)
	dynamo := awsclient.NewDynamo(ctx)
	snsClient := awsclient.NewSNS(ctx)
	proc := processor.New(dynamo, snsClient)

	// Health check server sobe antes de resolver a fila para o CI poder detectar o serviço
	port := getenv("PAYMENT_SERVICE_PORT", "3002")
	go startHealthServer(port)

	queueName := getenv("PAYMENT_QUEUE_NAME", "payment-queue-dev")
	queueURL := resolveQueueURL(ctx, sqsClient, queueName)

	logger.Info("payment-service consumer started", logger.Fields{"queue": queueName})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("Shutting down payment-service")
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		out, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
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

	// Extract trace propagation from message attributes
	parentSpanID := ""
	if attr, ok := msg.MessageAttributes["spanId"]; ok && attr.StringValue != nil {
		parentSpanID = *attr.StringValue
	}

	// Schema validation before unmarshalling — non-retryable if schema is wrong
	eventType, version := eventschema.ExtractMeta(raw)
	if err := eventschema.Validate(eventType, eventschema.Version(version), raw); err != nil {
		logger.Error("Schema validation failed — routing to DLQ", logger.Fields{
			"messageId": *msg.MessageId, "eventType": eventType, "error": err.Error(),
		})
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
		return
	}

	var event domain.OrderCreatedEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		logger.Error("Failed to parse message — routing to DLQ", logger.Fields{"messageId": *msg.MessageId})
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
		return
	}

	correlationID := event.Meta.CorrelationID

	// Open a child span, continuing the trace from the publisher
	ctx, span := tracing.StartWithTrace(ctx, correlationID, parentSpanID, "payment.process")
	span.Tag("orderId", event.OrderID).Tag("messageId", *msg.MessageId)
	defer func() {}() // span.Finish called after error check below

	err := proc.Process(ctx, event)
	if err == nil {
		span.Finish(nil)
		deleteMessage(ctx, sqsClient, queueURL, *msg.ReceiptHandle)
		logger.Info("Message processed", logger.Fields{"correlationId": correlationID, "messageId": *msg.MessageId})
		return
	}

	span.Finish(err)
	logger.Warn("Processing failed", logger.Fields{
		"correlationId": correlationID,
		"messageId":     *msg.MessageId,
		"receiveCount":  receiveCount,
		"error":         err.Error(),
	})

	if receiveCount >= 3 {
		// Exhaust maxReceiveCount → SQS routes to DLQ
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, 0)
	} else {
		backoff := int32(30 * receiveCount)
		if backoff > 300 {
			backoff = 300
		}
		changeVisibility(ctx, sqsClient, queueURL, *msg.ReceiptHandle, backoff)
	}
}

func resolveQueueURL(ctx context.Context, client *sqs.Client, name string) string {
	for {
		out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
		if err == nil {
			return *out.QueueUrl
		}
		logger.Warn("Queue not ready yet — retrying", logger.Fields{"queue": name, "error": err.Error()})
		time.Sleep(3 * time.Second)
	}
}

func deleteMessage(ctx context.Context, client *sqs.Client, url, receipt string) {
	_, _ = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: aws.String(url), ReceiptHandle: aws.String(receipt)})
}

func changeVisibility(ctx context.Context, client *sqs.Client, url, receipt string, timeout int32) {
	_, _ = client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl: aws.String(url), ReceiptHandle: aws.String(receipt), VisibilityTimeout: timeout,
	})
}

func startHealthServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","service":"payment-service"}`)
	})
	_ = http.ListenAndServe(":"+port, mux)
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
