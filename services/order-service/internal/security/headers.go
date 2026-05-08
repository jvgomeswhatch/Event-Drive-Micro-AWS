package security

import "net/http"

// SecureHeaders adds security response headers to every HTTP response.
// These protect against common web vulnerabilities at the transport layer.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Deny framing (clickjacking)
		w.Header().Set("X-Frame-Options", "DENY")
		// XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Strict transport security (enforced only in prod behind TLS)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Restrict referrer info
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Minimal permissions policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Remove server fingerprint
		w.Header().Del("Server")

		next.ServeHTTP(w, r)
	})
}
