package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCorrelationID_GeraSeAusente(t *testing.T) {
	handler := CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(CorrelationIDKey).(string)
		if id == "" {
			t.Error("esperava correlationId no contexto")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Correlation-ID") == "" {
		t.Error("esperava X-Correlation-ID no response header")
	}
}

func TestCorrelationID_PropagaExistente(t *testing.T) {
	idEsperado := "corr-abc-123"
	handler := CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(CorrelationIDKey).(string)
		if id != idEsperado {
			t.Errorf("correlationId: want %q, got %q", idEsperado, id)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Correlation-ID", idEsperado)
	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestAuth_SemJWTSecret_Retorna500(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	handler := Auth(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer qualquer.token.aqui")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("sem JWT_SECRET: want 500, got %d", w.Code)
	}
}

func TestAuth_SemHeader(t *testing.T) {
	handler := Auth(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_TokenInvalido(t *testing.T) {
	t.Setenv("JWT_SECRET", "qualquer-segredo-de-teste")
	handler := Auth(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer token.invalido.aqui")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestAuth_TokenValido(t *testing.T) {
	secret := "segredo-de-teste"
	os.Setenv("JWT_SECRET", secret)
	defer os.Unsetenv("JWT_SECRET")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "cust-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	handler := Auth(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestAuth_TokenExpirado(t *testing.T) {
	secret := "segredo-de-teste"
	os.Setenv("JWT_SECRET", secret)
	defer os.Unsetenv("JWT_SECRET")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "cust-1",
		"exp": time.Now().Add(-time.Hour).Unix(), // expirado
	})
	signed, _ := token.SignedString([]byte(secret))

	handler := Auth(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", w.Code)
	}
}

func TestCORS_OrigensPermitidas(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	handler := CORS(okHandler())

	// Origem permitida
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("origin permitida: want %q, got %q", "https://app.example.com", got)
	}

	// Origem não permitida
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Origin", "https://malicioso.com")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origem não permitida não deveria ter ACAO header, got %q", got)
	}
}

func TestCORS_SemRestricao(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	handler := CORS(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://qualquer.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://qualquer.com" {
		t.Errorf("sem restrição: want origem ecoada, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
	handler := CORS(okHandler())
	req := httptest.NewRequest(http.MethodOptions, "/orders", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight want 204, got %d", w.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
