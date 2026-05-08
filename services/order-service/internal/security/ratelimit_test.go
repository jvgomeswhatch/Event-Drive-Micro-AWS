package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 3) // 10 req/s, burst 3
	ip := "192.168.1.1"

	// As 3 primeiras devem passar (burst)
	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Errorf("request %d deveria ser permitida", i+1)
		}
	}
	// A quarta deve ser bloqueada (burst esgotado)
	if rl.Allow(ip) {
		t.Error("request 4 deveria ser bloqueada")
	}
}

func TestRateLimiter_RefillComTempo(t *testing.T) {
	rl := NewRateLimiter(100, 1) // 100 tok/s, burst 1
	ip := "10.0.0.1"

	rl.Allow(ip) // esgota burst

	time.Sleep(15 * time.Millisecond) // ≈1.5 tokens refil

	if !rl.Allow(ip) {
		t.Error("esperava token disponível após refil")
	}
}

func TestRateLimiter_IPsIndependentes(t *testing.T) {
	rl := NewRateLimiter(10, 1) // burst 1

	if !rl.Allow("1.1.1.1") {
		t.Error("IP 1.1.1.1 deveria ser permitido")
	}
	// IP diferente tem seu próprio bucket
	if !rl.Allow("2.2.2.2") {
		t.Error("IP 2.2.2.2 deveria ser permitido (bucket separado)")
	}
}

func TestRateLimiter_Middleware_Bloqueado(t *testing.T) {
	rl := NewRateLimiter(10, 0) // burst 0 — bloqueia tudo

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("want 429, got %d", w.Code)
	}
}

// Bug fix: goroutine de cleanup deve encerrar ao chamar Stop()
func TestRateLimiter_Stop_EncerraGoroutine(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	// Popula um bucket para garantir que cleanup tem trabalho
	rl.Allow("1.2.3.4")
	// Stop deve encerrar sem travar — se a goroutine não parar,
	// o teste vai vazar (detectado por go test -race ou leak detectors)
	rl.Stop()
}

func TestRateLimiter_Stop_MultiplasVezes(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	// Chamar Stop mais de uma vez não deve causar panic
	rl.Stop()
	rl.Stop()
}

func TestRealIP_Headers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req.Header.Set("X-Real-IP", "203.0.113.1")
	if got := realIP(req); got != "203.0.113.1" {
		t.Errorf("X-Real-IP: want 203.0.113.1, got %q", got)
	}

	req.Header.Del("X-Real-IP")
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	if got := realIP(req); got != "203.0.113.2" {
		t.Errorf("X-Forwarded-For: want 203.0.113.2, got %q", got)
	}

	req.Header.Del("X-Forwarded-For")
	req.RemoteAddr = "10.0.0.1:1234"
	if got := realIP(req); got != "10.0.0.1:1234" {
		t.Errorf("RemoteAddr: want 10.0.0.1:1234, got %q", got)
	}
}
