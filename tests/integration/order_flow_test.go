//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// TestFluxoOrdemCompleto verifies the full happy path:
//  1. Obtain JWT from order-service
//  2. POST /orders → 202 Accepted
//  3. order-service publishes order.created to SNS → fan-out reaches payment-queue
//  4. payment-service consumes and emits payment.succeeded or payment.failed
//
// All assertions use t.Fatal — no t.Skip.
func TestFluxoOrdemCompleto(t *testing.T) {
	ctx := context.Background()
	sqsCli := sqsClient(t)

	paymentURL := getQueueURL(ctx, t, sqsCli, envOu("PAYMENT_QUEUE_NAME", "payment-queue-dev"))
	purgeQueue(ctx, t, sqsCli, paymentURL)

	token := issueToken(t, "cust-integration")

	orderID := createOrder(t, token, map[string]any{
		"customerId":      "cust-integration",
		"simulateFailure": false,
		"items":           []map[string]any{{"productId": "prod-001", "quantity": 1}},
	}, "")
	t.Logf("order created: %s", orderID)

	// payment-service has up to 30s to consume order.created and publish its result.
	body, receipt := pollQueue(ctx, t, sqsCli, paymentURL, 30*time.Second)
	deleteMsg(ctx, sqsCli, paymentURL, receipt)

	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("could not parse payment event: %v — raw: %s", err, body)
	}

	et, _ := result["eventType"].(string)
	if et != "payment.succeeded" && et != "payment.failed" {
		t.Fatalf("expected eventType payment.succeeded or payment.failed, got %q", et)
	}
	t.Logf("payment event received: eventType=%s", et)
}

// TestDLQ_MensagemInvalida verifies that a message that fails schema validation
// is not silently dropped but eventually lands in the order DLQ.
// The message is sent directly to the order-queue (bypassing HTTP) so the
// order-service consumer receives and validates it.
func TestDLQ_MensagemInvalida(t *testing.T) {
	ctx := context.Background()
	sqsCli := sqsClient(t)

	dlqURL := getQueueURL(ctx, t, sqsCli, envOu("ORDER_DLQ_NAME", "order-dlq-dev"))
	purgeQueue(ctx, t, sqsCli, dlqURL)

	orderURL := getQueueURL(ctx, t, sqsCli, envOu("ORDER_QUEUE_NAME", "order-queue-dev"))

	// version "99" is unknown — eventschema.Validate will reject it.
	invalidPayload := fmt.Sprintf(
		`{"eventType":"order.created","version":"99","orderId":"dlq-test-%d","customerId":"cust-dlq","items":[]}`,
		time.Now().UnixNano(),
	)
	_, err := sqsCli.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(orderURL),
		MessageBody: aws.String(invalidPayload),
	})
	if err != nil {
		t.Fatalf("SendMessage to order-queue: %v", err)
	}

	// consumer sets visibility=0 on failure; after maxReceiveCount the message
	// is moved to the DLQ by SQS. Allow up to 45s for that cycle.
	body, receipt := pollQueue(ctx, t, sqsCli, dlqURL, 45*time.Second)
	deleteMsg(ctx, sqsCli, dlqURL, receipt)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("DLQ message is not valid JSON: %v — raw: %s", err, body)
	}
	if v, _ := parsed["version"].(string); v != "99" {
		t.Fatalf("expected the invalid message (version=99) in DLQ, got version=%q — raw: %s", v, body)
	}
	t.Logf("invalid message correctly routed to DLQ: %s", body)
}

// TestIdempotencia_MensagemDuplicada verifies that submitting the same order twice
// (same X-Idempotency-Key) results in at most one payment event — not two.
func TestIdempotencia_MensagemDuplicada(t *testing.T) {
	ctx := context.Background()
	sqsCli := sqsClient(t)

	paymentURL := getQueueURL(ctx, t, sqsCli, envOu("PAYMENT_QUEUE_NAME", "payment-queue-dev"))
	purgeQueue(ctx, t, sqsCli, paymentURL)

	token := issueToken(t, "cust-idem")
	idemKey := fmt.Sprintf("idem-test-%d", time.Now().UnixNano())

	orderBody := map[string]any{
		"customerId":      "cust-idem",
		"simulateFailure": false,
		"items":           []map[string]any{{"productId": "prod-001", "quantity": 1}},
	}

	// First request — accepted and processed normally.
	orderID1 := createOrder(t, token, orderBody, idemKey)
	t.Logf("first order: %s", orderID1)

	// Second request with the same idempotency key — the handler returns 200
	// {"cached":true} and must NOT publish another event.
	rawBody, _ := json.Marshal(orderBody)
	req2, _ := http.NewRequest(http.MethodPost, orderServiceURL()+"/orders", bytes.NewReader(rawBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("X-Idempotency-Key", idemKey)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("second POST /orders: %v", err)
	}
	resp2.Body.Close()
	t.Logf("duplicate request status: %d (expected 200 cached)", resp2.StatusCode)

	// Give payment-service time to process whatever was published.
	time.Sleep(15 * time.Second)

	count := countMessages(ctx, t, sqsCli, paymentURL)
	t.Logf("messages in payment-queue after duplicate order: %d", count)

	if count > 1 {
		t.Fatalf("idempotency failure: expected ≤1 payment event, got %d", count)
	}
}
