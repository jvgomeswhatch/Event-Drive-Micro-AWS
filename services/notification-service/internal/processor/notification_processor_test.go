package processor

import (
	"testing"

	"github.com/platform/notification-service/internal/domain"
)

func TestGetenv_ComValor(t *testing.T) {
	t.Setenv("TEST_NOTIF_KEY", "valor")
	if got := getenv("TEST_NOTIF_KEY", "fallback"); got != "valor" {
		t.Errorf("want 'valor', got %q", got)
	}
}

func TestGetenv_SemValor(t *testing.T) {
	if got := getenv("TEST_NOTIF_AUSENTE_XYZ", "fallback"); got != "fallback" {
		t.Errorf("want 'fallback', got %q", got)
	}
}

func TestNew_CamposInicializados(t *testing.T) {
	t.Setenv("ORDERS_TABLE", "orders-test")
	t.Setenv("EVENT_TIMELINE_TABLE", "timeline-test")
	t.Setenv("IDEMPOTENCY_TABLE", "idem-test")

	p := New(nil)
	if p.orders != "orders-test" {
		t.Errorf("orders: want 'orders-test', got %q", p.orders)
	}
	if p.timeline != "timeline-test" {
		t.Errorf("timeline: want 'timeline-test', got %q", p.timeline)
	}
	if p.idemTable != "idem-test" {
		t.Errorf("idemTable: want 'idem-test', got %q", p.idemTable)
	}
}

func TestBuildMessage_TodosOsTipos(t *testing.T) {
	casos := []struct {
		eventType string
		event     domain.IncomingEvent
		esperado  string
	}{
		{
			eventType: "payment.succeeded",
			event:     domain.IncomingEvent{EventType: "payment.succeeded", TotalAmount: 299.99},
			esperado:  "Your payment of $299.99 was successful.",
		},
		{
			eventType: "payment.failed",
			event:     domain.IncomingEvent{EventType: "payment.failed", FailureReason: "insufficient funds"},
			esperado:  "Payment failed: insufficient funds. Please retry.",
		},
		{
			eventType: "inventory.reserved",
			event:     domain.IncomingEvent{EventType: "inventory.reserved"},
			esperado:  "Your items have been reserved and your order is confirmed.",
		},
		{
			eventType: "inventory.failed",
			event:     domain.IncomingEvent{EventType: "inventory.failed", FailureReason: "out of stock"},
			esperado:  "Order could not be fulfilled: out of stock.",
		},
		{
			eventType: "unknown.event",
			event:     domain.IncomingEvent{EventType: "unknown.event"},
			esperado:  "Order update: unknown.event",
		},
	}

	for _, tc := range casos {
		got := buildMessage(tc.event)
		if got != tc.esperado {
			t.Errorf("buildMessage(%q): want %q, got %q", tc.eventType, tc.esperado, got)
		}
	}
}

func TestStatusMap_MapeamentoCorreto(t *testing.T) {
	esperados := map[string]string{
		"payment.succeeded":  "payment_confirmed",
		"payment.failed":     "payment_failed",
		"inventory.reserved": "confirmed",
		"inventory.failed":   "fulfillment_failed",
	}

	for eventType, statusEsperado := range esperados {
		got, ok := domain.StatusMap[eventType]
		if !ok {
			t.Errorf("StatusMap não contém eventType %q", eventType)
			continue
		}
		if got != statusEsperado {
			t.Errorf("StatusMap[%q]: want %q, got %q", eventType, statusEsperado, got)
		}
	}
}

func TestStatusMap_EventoDesconhecido(t *testing.T) {
	_, ok := domain.StatusMap["evento.desconhecido"]
	if ok {
		t.Error("StatusMap não deveria conter 'evento.desconhecido'")
	}
}
