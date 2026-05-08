package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/platform/order-service/internal/logger"
	"github.com/platform/order-service/internal/tracing"
)

type contextKey string

const CorrelationIDKey contextKey = "correlationId"
const UserKey contextKey = "user"

// CorrelationID extracts or generates a trace ID and opens a root span.
// Downstream handlers retrieve the span via tracing.FromContext(ctx).
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Correlation-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		parentSpanID := r.Header.Get("X-Span-ID")

		ctx := context.WithValue(r.Context(), CorrelationIDKey, traceID)
		ctx, _ = tracing.StartWithTrace(ctx, traceID, parentSpanID, r.Method+" "+r.URL.Path)

		w.Header().Set("X-Correlation-ID", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		corrID, _ := r.Context().Value(CorrelationIDKey).(string)
		span := tracing.FromContext(r.Context())

		fields := logger.Fields{
			"correlationId": corrID,
			"method":        r.Method,
			"path":          r.URL.Path,
			"status":        rw.status,
			"durationMs":    time.Since(start).Milliseconds(),
			"remoteAddr":    r.RemoteAddr,
		}
		if span != nil {
			fields["spanId"] = span.SpanID
		}
		logger.Info("HTTP request", fields)
	})
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing or malformed Authorization header"})
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server misconfiguration"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			corrID, _ := r.Context().Value(CorrelationIDKey).(string)
			logger.Warn("JWT verification failed", logger.Fields{"correlationId": corrID, "error": err.Error()})
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), UserKey, token.Claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORS aplica política de origem com base em CORS_ALLOWED_ORIGINS (env var, separado por vírgula).
// Em desenvolvimento, aceita qualquer origem quando a variável não for definida.
func CORS(next http.Handler) http.Handler {
	allowed := buildAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if isOriginAllowed(origin, allowed) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Correlation-ID,X-Idempotency-Key,X-Span-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func buildAllowedOrigins() []string {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if raw == "" {
		return nil // nil = permitir tudo (dev fallback)
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func isOriginAllowed(origin string, allowed []string) bool {
	if allowed == nil {
		return true
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
