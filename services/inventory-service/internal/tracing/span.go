// Package tracing provides lightweight in-process span tracking.
// It is NOT OpenTelemetry — it simulates trace/span IDs propagated via
// HTTP headers and SQS message attributes so logs can be correlated across
// services without a full tracing infrastructure.
//
// Header protocol:
//   X-Correlation-ID  — trace root, generated at API Gateway / first request
//   X-Span-ID         — current service span (generated per handler/consumer)
//   X-Parent-Span-ID  — span ID of the upstream caller
package tracing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/platform/inventory-service/internal/logger"
)

type contextKey string

const (
	spanKey contextKey = "span"
)

// Span represents a single unit of work within one service.
type Span struct {
	TraceID       string // == correlationId — the root identifier
	SpanID        string // unique to this unit of work
	ParentSpanID  string // empty for root spans
	Service       string
	Operation     string
	StartTime     time.Time
	tags          map[string]any
}

// Start creates a new child span from an existing trace context.
func Start(ctx context.Context, operation string) (context.Context, *Span) {
	parent, _ := ctx.Value(spanKey).(*Span)

	span := &Span{
		SpanID:    uuid.New().String()[:8],
		Service:   getenv("SERVICE_NAME", "unknown"),
		Operation: operation,
		StartTime: time.Now(),
		tags:      map[string]any{},
	}

	if parent != nil {
		span.TraceID      = parent.TraceID
		span.ParentSpanID = parent.SpanID
	} else {
		span.TraceID = uuid.New().String()
	}

	ctx = context.WithValue(ctx, spanKey, span)
	return ctx, span
}

// StartWithTrace creates a root span from a known correlationId (e.g. from HTTP header).
func StartWithTrace(ctx context.Context, traceID, parentSpanID, operation string) (context.Context, *Span) {
	span := &Span{
		TraceID:      traceID,
		SpanID:       uuid.New().String()[:8],
		ParentSpanID: parentSpanID,
		Service:      getenv("SERVICE_NAME", "unknown"),
		Operation:    operation,
		StartTime:    time.Now(),
		tags:         map[string]any{},
	}
	ctx = context.WithValue(ctx, spanKey, span)
	return ctx, span
}

// Tag adds a key-value annotation to the span.
func (s *Span) Tag(key string, value any) *Span {
	s.tags[key] = value
	return s
}

// Finish logs the span as a structured JSON entry.
func (s *Span) Finish(err error) {
	duration := time.Since(s.StartTime).Milliseconds()
	fields := logger.Fields{
		"traceId":      s.TraceID,
		"spanId":       s.SpanID,
		"parentSpanId": s.ParentSpanID,
		"service":      s.Service,
		"operation":    s.Operation,
		"durationMs":   duration,
	}
	for k, v := range s.tags {
		fields[k] = v
	}

	if err != nil {
		fields["error"] = err.Error()
		fields["status"] = "error"
		logger.Error("span finished", fields)
	} else {
		fields["status"] = "ok"
		logger.Info("span finished", fields)
	}
}

// FromContext returns the active span, or nil.
func FromContext(ctx context.Context) *Span {
	s, _ := ctx.Value(spanKey).(*Span)
	return s
}

// PropagationHeaders returns headers to forward to downstream services/queues.
func (s *Span) PropagationHeaders() map[string]string {
	return map[string]string{
		"X-Correlation-ID": s.TraceID,
		"X-Span-ID":        s.SpanID,
		"X-Parent-Span-ID": s.ParentSpanID,
	}
}

func getenv(k, fallback string) string {
	if v := envGet(k); v != "" {
		return v
	}
	return fallback
}
