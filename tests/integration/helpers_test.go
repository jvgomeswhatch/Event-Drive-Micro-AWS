//go:build integration

// Integration tests — require LocalStack + services running (make up).
// Run with: go test -mod=vendor -v -tags=integration -timeout 120s ./...
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func localstackEndpoint() string {
	if ep := os.Getenv("LOCALSTACK_ENDPOINT"); ep != "" {
		return ep
	}
	return "http://localhost:4566"
}

func orderServiceURL() string {
	if u := os.Getenv("ORDER_SERVICE_URL"); u != "" {
		return u
	}
	return "http://localhost:3001"
}

func awsCfg(t *testing.T) aws.Config {
	t.Helper()
	endpoint := localstackEndpoint()
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint, HostnameImmutable: true}, nil
			},
		)),
	)
	if err != nil {
		t.Fatalf("aws config: %v", err)
	}
	return cfg
}

func sqsClient(t *testing.T) *sqs.Client {
	t.Helper()
	return sqs.NewFromConfig(awsCfg(t))
}

func dynamoClient(t *testing.T) *dynamodb.Client {
	t.Helper()
	return dynamodb.NewFromConfig(awsCfg(t))
}

func envOu(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getQueueURL resolves the SQS queue URL, failing the test (not skipping) if unavailable.
func getQueueURL(ctx context.Context, t *testing.T, client *sqs.Client, name string) string {
	t.Helper()
	out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
	if err != nil {
		t.Fatalf("queue %q not found — is LocalStack running? %v", name, err)
	}
	return *out.QueueUrl
}

// issueToken calls POST /auth/token and returns a Bearer JWT.
func issueToken(t *testing.T, customerID string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"customerId": customerID, "name": "Integration Test"})
	resp, err := http.Post(orderServiceURL()+"/auth/token", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/token: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /auth/token status %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.Token == "" {
		t.Fatalf("could not parse token response: %s", raw)
	}
	return out.Token
}

// createOrder calls POST /orders and returns the orderId from the response.
func createOrder(t *testing.T, token string, body map[string]any, idempotencyKey string) string {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, orderServiceURL()+"/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if idempotencyKey != "" {
		req.Header.Set("X-Idempotency-Key", idempotencyKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /orders: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /orders expected 202, got %d: %s", resp.StatusCode, respBody)
	}

	var out struct {
		OrderID string `json:"orderId"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil || out.OrderID == "" {
		t.Fatalf("could not parse orderId from response: %s", respBody)
	}
	return out.OrderID
}

// pollQueue polls an SQS queue until a message arrives or the timeout elapses.
// Returns the message body and its receipt handle (so the caller can delete it).
// Fails the test (not skips) if nothing arrives within the timeout.
func pollQueue(ctx context.Context, t *testing.T, client *sqs.Client, queueURL string, timeout time.Duration) (body string, receipt string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     2,
		})
		if err != nil {
			t.Logf("ReceiveMessage: %v", err)
			continue
		}
		if len(out.Messages) > 0 {
			return *out.Messages[0].Body, *out.Messages[0].ReceiptHandle
		}
	}
	t.Fatalf("no message arrived in %s on queue %s", timeout, queueURL)
	return "", ""
}

// deleteMsg removes a message from a queue after it has been consumed.
func deleteMsg(ctx context.Context, client *sqs.Client, queueURL, receipt string) {
	_, _ = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receipt),
	})
}

// countMessages returns the approximate message count in a queue.
func countMessages(ctx context.Context, t *testing.T, client *sqs.Client, queueURL string) int {
	t.Helper()
	out, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameApproximateNumberOfMessages},
	})
	if err != nil {
		t.Fatalf("GetQueueAttributes: %v", err)
	}
	count := 0
	fmt.Sscanf(out.Attributes["ApproximateNumberOfMessages"], "%d", &count)
	return count
}

// purgeQueue drains a queue and waits briefly for SQS to propagate the purge.
func purgeQueue(ctx context.Context, t *testing.T, client *sqs.Client, queueURL string) {
	t.Helper()
	_, err := client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: aws.String(queueURL)})
	if err != nil {
		t.Logf("PurgeQueue warning: %v", err)
	}
	time.Sleep(2 * time.Second)
}
