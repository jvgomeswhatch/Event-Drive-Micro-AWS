package security

import (
	"testing"
)

func TestSanitizeString(t *testing.T) {
	casos := []struct {
		entrada string
		esperado string
	}{
		{"  olá  ", "olá"},
		{"texto\x00nulo", "textonulo"},
		{"texto\nnova linha", "texto\nnova linha"}, // newline preservado
		{"texto\ttab", "texto\ttab"},               // tab preservado
		{"\x01\x02\x03", ""},
	}

	for _, tc := range casos {
		got := SanitizeString(tc.entrada)
		if got != tc.esperado {
			t.Errorf("SanitizeString(%q) = %q, want %q", tc.entrada, got, tc.esperado)
		}
	}
}

func TestSanitizeID(t *testing.T) {
	casos := []struct {
		entrada  string
		wantOK   bool
		wantVal  string
	}{
		{"cliente-001", true, "cliente-001"},
		{"prod_abc_123", true, "prod_abc_123"},
		{"uuid-com-tracos", true, "uuid-com-tracos"},
		{"espaço com acento", false, ""}, // espaço interno não é permitido
		{"injeção'; DROP TABLE", false, ""},
		{"../etc/passwd", false, ""},
		{"", true, ""},
	}

	for _, tc := range casos {
		got, ok := SanitizeID(tc.entrada)
		if ok != tc.wantOK {
			t.Errorf("SanitizeID(%q) ok = %v, want %v", tc.entrada, ok, tc.wantOK)
		}
		if ok && got != tc.wantVal {
			t.Errorf("SanitizeID(%q) val = %q, want %q", tc.entrada, got, tc.wantVal)
		}
	}
}

func TestIsSafeLogValue(t *testing.T) {
	if !IsSafeLogValue("valor normal") {
		t.Error("esperava true para valor normal")
	}
	if IsSafeLogValue("valor\ncom newline") {
		t.Error("esperava false para valor com newline")
	}
	if IsSafeLogValue("valor\rcom carriage return") {
		t.Error("esperava false para valor com carriage return")
	}
}

func TestSafeLogString(t *testing.T) {
	entrada := "linha1\nlinha2\rretorno"
	got := SafeLogString(entrada)
	esperado := "linha1\\nlinha2\\rretorno"
	if got != esperado {
		t.Errorf("SafeLogString = %q, want %q", got, esperado)
	}
}
