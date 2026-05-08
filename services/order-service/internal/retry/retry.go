package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/platform/order-service/internal/logger"
)

type NonRetryableError struct {
	Cause error
}

func (e *NonRetryableError) Error() string { return e.Cause.Error() }
func (e *NonRetryableError) Unwrap() error { return e.Cause }

// NonRetryable wraps an error to signal it must not be retried.
func NonRetryable(err error) error { return &NonRetryableError{Cause: err} }

// Do runs fn with exponential backoff. Stops immediately on NonRetryableError.
func Do(ctx context.Context, maxAttempts int, baseDelay time.Duration, correlationID string, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		var nre *NonRetryableError
		if isNonRetryable(err, &nre) {
			logger.Warn("Non-retryable error — aborting", logger.Fields{
				"correlationId": correlationID,
				"attempt":       attempt,
				"error":         err.Error(),
			})
			return err
		}

		lastErr = err
		if attempt == maxAttempts {
			break
		}

		jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
		delay := time.Duration(math.Pow(2, float64(attempt-1)))*baseDelay + jitter

		logger.Warn("Retryable error — backing off", logger.Fields{
			"correlationId": correlationID,
			"attempt":       attempt,
			"delayMs":       delay.Milliseconds(),
			"error":         err.Error(),
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

func isNonRetryable(err error, target **NonRetryableError) bool {
	if err == nil {
		return false
	}
	// Walk the error chain
	e := err
	for e != nil {
		if nre, ok := e.(*NonRetryableError); ok {
			*target = nre
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := e.(unwrapper); ok {
			e = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
