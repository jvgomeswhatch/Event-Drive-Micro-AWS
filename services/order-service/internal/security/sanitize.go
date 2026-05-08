package security

import (
	"strings"
	"unicode"
)

// SanitizeString removes control characters and trims whitespace.
// It does NOT HTML-escape — that is the responsibility of the output layer.
// Use at input boundaries: request body fields, query params, path params.
func SanitizeString(s string) string {
	// Strip non-printable control characters (except newline/tab which are harmless in logs)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(cleaned)
}

// SanitizeID sanitizes an identifier field (customerId, productId, orderId).
// Allows only alphanumeric characters, hyphens, and underscores.
// Returns ("", false) if the value contains invalid characters.
func SanitizeID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return "", false
		}
	}
	return s, true
}

// IsSafeLogValue checks that a value being logged does not contain
// newline injection sequences that could corrupt structured log streams.
func IsSafeLogValue(s string) bool {
	return !strings.ContainsAny(s, "\n\r")
}

// SafeLogString escapes newlines in a value before logging.
func SafeLogString(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}
