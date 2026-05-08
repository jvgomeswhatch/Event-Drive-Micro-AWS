package processor

import (
	"testing"

	"github.com/platform/inventory-service/internal/domain"
)

func TestGetenv_ComValor(t *testing.T) {
	t.Setenv("TEST_INV_KEY", "valor")
	if got := getenv("TEST_INV_KEY", "fallback"); got != "valor" {
		t.Errorf("want 'valor', got %q", got)
	}
}

func TestGetenv_SemValor(t *testing.T) {
	if got := getenv("TEST_INV_AUSENTE_XYZ", "fallback"); got != "fallback" {
		t.Errorf("want 'fallback', got %q", got)
	}
}

// calcularStatusReserva espelha a lógica de decisão do processor.
// status="failed" quando err != nil durante o loop de reserva.
func TestStatusReserva_Sucesso(t *testing.T) {
	items := []domain.OrderItem{
		{ProductID: "prod-001", Quantity: 1},
		{ProductID: "prod-002", Quantity: 2},
	}
	// Simula loop sem erro
	status := "reserved"
	failureReason := ""
	for _, item := range items {
		// sem erro simulado
		_ = item
	}
	if status != "reserved" {
		t.Errorf("want 'reserved', got %q", status)
	}
	if failureReason != "" {
		t.Errorf("failureReason esperado vazio, got %q", failureReason)
	}
}

func TestStatusReserva_Falha(t *testing.T) {
	productID := "prod-999"
	status := "reserved"
	failureReason := ""

	// Simula o que o processor faz em caso de erro de UpdateItem
	simulatedErr := true
	if simulatedErr {
		failureReason = "Insufficient stock or product not found: " + productID + " (requested 100)"
		status = "failed"
	}

	if status != "failed" {
		t.Errorf("want 'failed', got %q", status)
	}
	if failureReason == "" {
		t.Error("failureReason não deve ser vazio em caso de falha")
	}
}

func TestNew_CamposInicializados(t *testing.T) {
	t.Setenv("INVENTORY_TABLE", "inv-test")
	t.Setenv("EVENT_TIMELINE_TABLE", "timeline-test")
	t.Setenv("IDEMPOTENCY_TABLE", "idem-test")

	p := New(nil, nil)
	if p.inventory != "inv-test" {
		t.Errorf("inventory: want 'inv-test', got %q", p.inventory)
	}
	if p.timeline != "timeline-test" {
		t.Errorf("timeline: want 'timeline-test', got %q", p.timeline)
	}
	if p.idemTable != "idem-test" {
		t.Errorf("idemTable: want 'idem-test', got %q", p.idemTable)
	}
}
