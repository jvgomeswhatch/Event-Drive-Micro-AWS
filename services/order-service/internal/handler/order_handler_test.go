package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/platform/order-service/internal/domain"
)

// ── validateCreateRequest ────────────────────────────────────────────────────

func TestValidateCreateRequest(t *testing.T) {
	validItems := []domain.OrderItem{{ProductID: "prod-001", Quantity: 2}}

	casos := []struct {
		nome    string
		req     domain.CreateOrderRequest
		wantErr bool
		msgErro string
	}{
		{
			nome:    "requisição válida",
			req:     domain.CreateOrderRequest{CustomerID: "cust-1", Items: validItems},
			wantErr: false,
		},
		{
			nome:    "customerId vazio",
			req:     domain.CreateOrderRequest{CustomerID: "   ", Items: validItems},
			wantErr: true,
			msgErro: "customerId is required",
		},
		{
			nome:    "sem itens",
			req:     domain.CreateOrderRequest{CustomerID: "cust-1", Items: []domain.OrderItem{}},
			wantErr: true,
			msgErro: "items must not be empty",
		},
		{
			nome:    "itens excedem 50",
			req:     domain.CreateOrderRequest{CustomerID: "cust-1", Items: makeItems(51)},
			wantErr: true,
			msgErro: "items must not exceed 50",
		},
		{
			nome: "productId vazio",
			req: domain.CreateOrderRequest{
				CustomerID: "cust-1",
				Items:      []domain.OrderItem{{ProductID: "", Quantity: 1}},
			},
			wantErr: true,
			msgErro: "item productId is required",
		},
		{
			nome: "quantity zero",
			req: domain.CreateOrderRequest{
				CustomerID: "cust-1",
				Items:      []domain.OrderItem{{ProductID: "p1", Quantity: 0}},
			},
			wantErr: true,
			msgErro: "item quantity must be between 1 and 1000",
		},
		{
			nome: "quantity acima de 1000",
			req: domain.CreateOrderRequest{
				CustomerID: "cust-1",
				Items:      []domain.OrderItem{{ProductID: "p1", Quantity: 1001}},
			},
			wantErr: true,
			msgErro: "item quantity must be between 1 and 1000",
		},
		{
			nome: "quantity no limite superior",
			req: domain.CreateOrderRequest{
				CustomerID: "cust-1",
				Items:      []domain.OrderItem{{ProductID: "p1", Quantity: 1000}},
			},
			wantErr: false,
		},
		{
			nome:    "máximo de 50 itens",
			req:     domain.CreateOrderRequest{CustomerID: "cust-1", Items: makeItems(50)},
			wantErr: false,
		},
	}

	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			err := validateCreateRequest(tc.req)
			if tc.wantErr && err == nil {
				t.Error("esperava erro, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("esperava nil, got: %v", err)
			}
			if tc.wantErr && err != nil && tc.msgErro != "" && err.Error() != tc.msgErro {
				t.Errorf("mensagem de erro: want %q, got %q", tc.msgErro, err.Error())
			}
		})
	}
}

// ── UUID regex ───────────────────────────────────────────────────────────────

func TestUUIDRegex(t *testing.T) {
	validos := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479",
	}
	invalidos := []string{
		"",
		"../etc/passwd",
		"not-a-uuid",
		"550e8400e29b41d4a716446655440000", // sem hifens
		"550e8400-e29b-41d4-a716-44665544000g", // char inválido
	}

	for _, id := range validos {
		if !uuidRe.MatchString(id) {
			t.Errorf("esperava UUID válido: %q", id)
		}
	}
	for _, id := range invalidos {
		if uuidRe.MatchString(id) {
			t.Errorf("esperava UUID inválido: %q", id)
		}
	}
}

// ── jsonError / jsonResponse ─────────────────────────────────────────────────

func TestJsonError(t *testing.T) {
	w := httptest.NewRecorder()
	jsonError(w, "algum erro", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["error"] != "algum erro" {
		t.Errorf("body[error]: want %q, got %q", "algum erro", body["error"])
	}
}

func TestJsonResponse(t *testing.T) {
	w := httptest.NewRecorder()
	jsonResponse(w, map[string]string{"ok": "sim"}, http.StatusAccepted)

	if w.Code != http.StatusAccepted {
		t.Errorf("status: want 202, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
}

// ── Create — validação via HTTP ──────────────────────────────────────────────

func TestCreateHandler_InvalidJSON(t *testing.T) {
	h := &OrderHandler{}
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("not json"))
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestCreateHandler_MissingCustomerID(t *testing.T) {
	h := &OrderHandler{}
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{{"productId": "p1", "quantity": 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ── IssueToken — Bug fix: JWT_SECRET ausente deve retornar 500 ───────────────
//
// Bug anterior: auth_handler usava fallback "change-me-in-production" quando
// JWT_SECRET não estava definido, enquanto o middleware Auth retornava 500
// nessa mesma situação. O token era emitido mas nunca podia ser validado.
// Correção: ambos agora retornam 500 sem JWT_SECRET, comportamento consistente.

func TestIssueToken_SemJWTSecret_Retorna500(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	h := NewAuthHandler()
	body, _ := json.Marshal(map[string]string{"customerId": "user-123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.IssueToken(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("sem JWT_SECRET: want 500, got %d", w.Code)
	}
}

func TestIssueToken_ComJWTSecret_Retorna200(t *testing.T) {
	t.Setenv("JWT_SECRET", "segredo-de-teste-valido")
	h := NewAuthHandler()
	body, _ := json.Marshal(map[string]string{"customerId": "user-123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.IssueToken(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("com JWT_SECRET: want 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["token"] == "" {
		t.Error("esperava campo 'token' no response")
	}
}

func TestIssueToken_SemCustomerID_Retorna400(t *testing.T) {
	t.Setenv("JWT_SECRET", "segredo-de-teste-valido")
	h := NewAuthHandler()
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.IssueToken(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("sem customerId: want 400, got %d", w.Code)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeItems(n int) []domain.OrderItem {
	items := make([]domain.OrderItem, n)
	for i := range items {
		items[i] = domain.OrderItem{ProductID: "p", Quantity: 1}
	}
	return items
}
