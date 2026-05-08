package eventschema

import (
	"encoding/json"
	"testing"
)

func TestValidate_OrderCreated_Valid(t *testing.T) {
	payload := map[string]any{
		"eventType":  "order.created",
		"version":    "1",
		"orderId":    "550e8400-e29b-41d4-a716-446655440000",
		"customerId": "cust-1",
		"items":      []map[string]any{{"productId": "p1", "quantity": 2}},
		"_meta": map[string]any{
			"correlationId": "corr-1",
			"publishedAt":   "2026-01-01T00:00:00Z",
			"publisher":     "order-service",
		},
	}
	raw, _ := json.Marshal(payload)
	if err := Validate("order.created", V1, raw); err != nil {
		t.Fatalf("expected valid, got error: %v", err)
	}
}

func TestValidate_OrderCreated_MissingField(t *testing.T) {
	payload := map[string]any{
		"eventType": "order.created",
		"version":   "1",
		"orderId":   "550e8400-e29b-41d4-a716-446655440000",
		// missing customerId, items, _meta
	}
	raw, _ := json.Marshal(payload)
	err := Validate("order.created", V1, raw)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(ve.Missing) == 0 {
		t.Error("expected missing fields to be non-empty")
	}
}

func TestValidate_UnknownEventType_PassThrough(t *testing.T) {
	// Unknown event types must pass — forward compatibility
	raw := []byte(`{"eventType":"future.event","version":"2","someField":"x"}`)
	if err := Validate("future.event", "2", raw); err != nil {
		t.Fatalf("unknown event type should pass through, got: %v", err)
	}
}

func TestValidate_UnknownVersion_Reject(t *testing.T) {
	raw := []byte(`{"eventType":"order.created","version":"99"}`)
	err := Validate("order.created", "99", raw)
	if err == nil {
		t.Fatal("expected error for unknown version, got nil")
	}
}

func TestExtractMeta(t *testing.T) {
	raw := []byte(`{"eventType":"order.created","version":"1","orderId":"x"}`)
	et, v := ExtractMeta(raw)
	if et != "order.created" {
		t.Errorf("expected eventType 'order.created', got %q", et)
	}
	if v != "1" {
		t.Errorf("expected version '1', got %q", v)
	}
}

func TestExtractMeta_MalformedJSON(t *testing.T) {
	et, v := ExtractMeta([]byte(`not json`))
	if et != "" || v != "" {
		t.Error("expected empty strings for malformed JSON")
	}
}
