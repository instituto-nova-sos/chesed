package middleware

import "net/http"

// hstsValue is the Strict-Transport-Security policy applied when HSTS is
// enabled: two years, subdomains included, eligible for the preload list.
const hstsValue = "max-age=63072000; includeSubDomains; preload"

// SecurityHeaders returns middleware that sets standard security headers
// without HSTS. Kept for back-compat with routes wired before the public API.
func SecurityHeaders(next http.Handler) http.Handler {
	return SecurityHeadersWith(false)(next)
}

// SecurityHeadersWith returns middleware that sets standard security headers.
// When hstsEnabled is true it additionally sets Strict-Transport-Security,
// which must only be enabled behind HTTPS (production/edge TLS).
func SecurityHeadersWith(hstsEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setBaseSecurityHeaders(w.Header())
			if hstsEnabled {
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// setBaseSecurityHeaders writes the headers shared by every variant.
func setBaseSecurityHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	h.Set("Cache-Control", "no-store")
}
