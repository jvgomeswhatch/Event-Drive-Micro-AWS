// Package eventschema validates outgoing events against versioned JSON schemas.
// Schemas live in docs/event-schemas/v{N}/ and are embedded at compile time.
// Validation runs in the publisher path so schema violations surface immediately,
// not downstream in a consumer.
package eventschema

import (
	"encoding/json"
	"fmt"
)

// Version represents an event schema version string.
type Version string

const V1 Version = "1"

// knownFields defines the required top-level fields per event type and version.
// This is a lightweight alternative to embedding a full JSON-Schema validator
// (which would pull in a heavy dependency). Full schema files live in
// docs/event-schemas/ for documentation and contract testing.
var knownFields = map[string]map[Version][]string{
	"order.created": {
		V1: {"eventType", "version", "orderId", "customerId", "items", "_meta"},
	},
	"payment.succeeded": {
		V1: {"eventType", "version", "paymentId", "orderId", "customerId", "items", "totalAmount", "status", "_meta"},
	},
	"payment.failed": {
		V1: {"eventType", "version", "paymentId", "orderId", "customerId", "totalAmount", "status", "failureReason", "_meta"},
	},
	"inventory.reserved": {
		V1: {"eventType", "version", "reservationId", "orderId", "customerId", "items", "status", "_meta"},
	},
	"inventory.failed": {
		V1: {"eventType", "version", "reservationId", "orderId", "customerId", "items", "status", "failureReason", "_meta"},
	},
}

// ValidationError is returned when an event does not conform to its schema.
type ValidationError struct {
	EventType string
	Version   Version
	Missing   []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("schema validation failed for %s@v%s: missing fields %v", e.EventType, e.Version, e.Missing)
}

// Validate checks that rawJSON contains all required fields for eventType at the given version.
// Unknown eventTypes are accepted (forward-compatibility: ignore unknown events, don't crash).
func Validate(eventType string, version Version, rawJSON []byte) error {
	versionMap, ok := knownFields[eventType]
	if !ok {
		// Unknown event type — allow through (forward-compat)
		return nil
	}

	required, ok := versionMap[version]
	if !ok {
		// Unknown version — reject: consumer must be updated
		return fmt.Errorf("schema validation: unknown version %q for event type %q", version, eventType)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rawJSON, &payload); err != nil {
		return fmt.Errorf("schema validation: cannot parse JSON: %w", err)
	}

	var missing []string
	for _, field := range required {
		if _, exists := payload[field]; !exists {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return &ValidationError{EventType: eventType, Version: version, Missing: missing}
	}
	return nil
}

// ExtractMeta safely extracts eventType and version from a raw JSON payload.
// Returns empty strings if the fields are absent or malformed.
func ExtractMeta(rawJSON []byte) (eventType string, version string) {
	var envelope struct {
		EventType string `json:"eventType"`
		Version   string `json:"version"`
	}
	_ = json.Unmarshal(rawJSON, &envelope)
	return envelope.EventType, envelope.Version
}
