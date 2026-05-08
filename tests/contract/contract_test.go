// Package contract validates that example event payloads conform to their
// published JSON schemas. Run with: go test ./tests/contract/...
package contract

import (
	"encoding/json"
	"fmt"
	"testing"
)

// knownFields mirrors the eventschema package — kept separate so contract tests
// don't depend on service internals.
var requiredFields = map[string]map[string][]string{
	"order.created": {
		"1": {"eventType", "version", "orderId", "customerId", "items", "_meta"},
	},
	"payment.succeeded": {
		"1": {"eventType", "version", "paymentId", "orderId", "customerId", "items", "totalAmount", "status", "_meta"},
	},
	"payment.failed": {
		"1": {"eventType", "version", "paymentId", "orderId", "customerId", "totalAmount", "status", "failureReason", "_meta"},
	},
	"inventory.reserved": {
		"1": {"eventType", "version", "reservationId", "orderId", "customerId", "items", "status", "_meta"},
	},
	"inventory.failed": {
		"1": {"eventType", "version", "reservationId", "orderId", "customerId", "items", "status", "failureReason", "_meta"},
	},
}

type contractCase struct {
	name      string
	eventType string
	version   string
	payload   map[string]any
	wantValid bool
}

var meta = map[string]any{
	"correlationId": "corr-abc-123",
	"publishedAt":   "2026-05-05T12:00:00Z",
	"publisher":     "order-service",
}

var paymentMeta = map[string]any{
	"correlationId": "corr-abc-123",
	"publishedAt":   "2026-05-05T12:00:00Z",
	"publisher":     "payment-service",
}

var inventoryMeta = map[string]any{
	"correlationId": "corr-abc-123",
	"publishedAt":   "2026-05-05T12:00:00Z",
	"publisher":     "inventory-service",
}

var items = []map[string]any{{"productId": "prod-001", "quantity": 2}}

var cases = []contractCase{
	// ── order.created ────────────────────────────────────────────────────────
	{
		name: "order.created v1 valid",
		eventType: "order.created", version: "1",
		payload: map[string]any{
			"eventType": "order.created", "version": "1",
			"orderId": "550e8400-e29b-41d4-a716-446655440000",
			"customerId": "cust-1", "items": items,
			"simulateFailure": false, "_meta": meta,
		},
		wantValid: true,
	},
	{
		name: "order.created v1 missing customerId",
		eventType: "order.created", version: "1",
		payload: map[string]any{
			"eventType": "order.created", "version": "1",
			"orderId": "550e8400-e29b-41d4-a716-446655440000",
			"items": items, "_meta": meta,
		},
		wantValid: false,
	},
	// ── payment.succeeded ────────────────────────────────────────────────────
	{
		name: "payment.succeeded v1 valid",
		eventType: "payment.succeeded", version: "1",
		payload: map[string]any{
			"eventType": "payment.succeeded", "version": "1",
			"paymentId": "pay-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "items": items,
			"totalAmount": 29.99, "status": "succeeded",
			"_meta": paymentMeta,
		},
		wantValid: true,
	},
	{
		name: "payment.succeeded v1 missing totalAmount",
		eventType: "payment.succeeded", version: "1",
		payload: map[string]any{
			"eventType": "payment.succeeded", "version": "1",
			"paymentId": "pay-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "items": items, "status": "succeeded",
			"_meta": paymentMeta,
		},
		wantValid: false,
	},
	// ── payment.failed ───────────────────────────────────────────────────────
	{
		name: "payment.failed v1 valid",
		eventType: "payment.failed", version: "1",
		payload: map[string]any{
			"eventType": "payment.failed", "version": "1",
			"paymentId": "pay-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "totalAmount": 29.99,
			"status": "failed", "failureReason": "Insufficient funds",
			"_meta": paymentMeta,
		},
		wantValid: true,
	},
	{
		name: "payment.failed v1 missing failureReason",
		eventType: "payment.failed", version: "1",
		payload: map[string]any{
			"eventType": "payment.failed", "version": "1",
			"paymentId": "pay-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "totalAmount": 29.99, "status": "failed",
			"_meta": paymentMeta,
		},
		wantValid: false,
	},
	// ── inventory.reserved ───────────────────────────────────────────────────
	{
		name: "inventory.reserved v1 valid",
		eventType: "inventory.reserved", version: "1",
		payload: map[string]any{
			"eventType": "inventory.reserved", "version": "1",
			"reservationId": "res-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "items": items, "status": "reserved",
			"_meta": inventoryMeta,
		},
		wantValid: true,
	},
	// ── inventory.failed ─────────────────────────────────────────────────────
	{
		name: "inventory.failed v1 valid",
		eventType: "inventory.failed", version: "1",
		payload: map[string]any{
			"eventType": "inventory.failed", "version": "1",
			"reservationId": "res-uuid", "orderId": "order-uuid",
			"customerId": "cust-1", "items": items,
			"status": "failed", "failureReason": "Insufficient stock for prod-001",
			"_meta": inventoryMeta,
		},
		wantValid: true,
	},
	// ── forward compatibility ─────────────────────────────────────────────────
	{
		name: "unknown eventType passes through",
		eventType: "future.event", version: "2",
		payload:   map[string]any{"eventType": "future.event", "version": "2"},
		wantValid: true,
	},
	{
		name: "known eventType with unknown version rejected",
		eventType: "order.created", version: "99",
		payload:   map[string]any{"eventType": "order.created", "version": "99"},
		wantValid: false,
	},
}

func validate(eventType, version string, raw []byte) error {
	versionMap, ok := requiredFields[eventType]
	if !ok {
		return nil // unknown type — forward compat
	}
	required, ok := versionMap[version]
	if !ok {
		return fmt.Errorf("unknown version %q for event type %q", version, eventType)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	var missing []string
	for _, f := range required {
		if _, exists := payload[f]; !exists {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}

func TestEventContracts(t *testing.T) {
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}
			err = validate(tc.eventType, tc.version, raw)
			if tc.wantValid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tc.wantValid && err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
