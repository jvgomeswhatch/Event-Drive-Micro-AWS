package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_Success(t *testing.T) {
	chamadas := 0
	err := Do(context.Background(), 3, time.Millisecond, "corr-1", func() error {
		chamadas++
		return nil
	})
	if err != nil {
		t.Fatalf("esperava nil, got %v", err)
	}
	if chamadas != 1 {
		t.Errorf("esperava 1 chamada, got %d", chamadas)
	}
}

func TestDo_RetryavelEExausto(t *testing.T) {
	chamadas := 0
	errBase := errors.New("erro transitório")
	err := Do(context.Background(), 3, time.Millisecond, "corr-1", func() error {
		chamadas++
		return errBase
	})
	if err == nil {
		t.Fatal("esperava erro, got nil")
	}
	if chamadas != 3 {
		t.Errorf("esperava 3 tentativas, got %d", chamadas)
	}
}

func TestDo_NaoRetryavel(t *testing.T) {
	chamadas := 0
	err := Do(context.Background(), 5, time.Millisecond, "corr-1", func() error {
		chamadas++
		return NonRetryable(errors.New("schema inválido"))
	})
	if err == nil {
		t.Fatal("esperava erro, got nil")
	}
	if chamadas != 1 {
		t.Errorf("esperava 1 chamada (parou imediatamente), got %d", chamadas)
	}
}

func TestDo_SuccessNaSegundaTentativa(t *testing.T) {
	chamadas := 0
	err := Do(context.Background(), 3, time.Millisecond, "corr-1", func() error {
		chamadas++
		if chamadas < 2 {
			return errors.New("temporário")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("esperava nil, got %v", err)
	}
	if chamadas != 2 {
		t.Errorf("esperava 2 chamadas, got %d", chamadas)
	}
}

func TestDo_ContextCancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelado imediatamente

	err := Do(ctx, 5, time.Second, "corr-1", func() error {
		return errors.New("erro")
	})
	// Primeira tentativa executa, depois ctx.Done() interrompe o backoff
	if err == nil {
		t.Fatal("esperava erro de contexto, got nil")
	}
}

func TestNonRetryable_Unwrap(t *testing.T) {
	causa := errors.New("causa raiz")
	wrapped := NonRetryable(causa)

	nre := wrapped.(*NonRetryableError)
	if !errors.Is(wrapped, causa) {
		t.Error("errors.Is deve atravessar NonRetryableError")
	}
	if nre.Error() != "causa raiz" {
		t.Errorf("Error() want %q, got %q", "causa raiz", nre.Error())
	}
}
