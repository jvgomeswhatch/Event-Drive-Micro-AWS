// Package security provides HTTP-boundary protections: rate limiting,
// input sanitization, and security headers.
// Rate limiter uses a token-bucket per IP with a sliding window.
package security

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/platform/order-service/internal/logger"
)

type bucket struct {
	tokens    float64
	lastRefil time.Time
	mu        sync.Mutex
}

type RateLimiter struct {
	buckets    map[string]*bucket
	mu         sync.RWMutex
	rate       float64 // tokens per second
	burst      float64 // max burst size
	cleanupTTL time.Duration
	stop       context.CancelFunc
}

func NewRateLimiter(requestsPerSecond, burst float64) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		buckets:    make(map[string]*bucket),
		rate:       requestsPerSecond,
		burst:      burst,
		cleanupTTL: 5 * time.Minute,
		stop:       cancel,
	}
	go rl.cleanup(ctx)
	return rl
}

// Stop encerra a goroutine de cleanup. Deve ser chamado quando o RateLimiter não for mais usado.
func (rl *RateLimiter) Stop() {
	rl.stop()
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.RLock()
	b, exists := rl.buckets[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		b = &bucket{tokens: rl.burst, lastRefil: time.Now()}
		rl.buckets[ip] = b
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefil).Seconds()
	b.tokens = minFloat(rl.burst, b.tokens+elapsed*rl.rate)
	b.lastRefil = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.cleanupTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, b := range rl.buckets {
				b.mu.Lock()
				if time.Since(b.lastRefil) > rl.cleanupTTL {
					delete(rl.buckets, ip)
				}
				b.mu.Unlock()
			}
			rl.mu.Unlock()
		}
	}
}

// Middleware returns an HTTP middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		if !rl.Allow(ip) {
			logger.Warn("Rate limit exceeded", logger.Fields{
				"ip":   ip,
				"path": r.URL.Path,
			})
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
