package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// PublicRateLimit throttles unauthenticated public API traffic per client IP,
// allowing requestsPerMinute requests in a rolling one-minute window. When the
// budget is exhausted it responds 429 with the project's JSON error shape and
// the Retry-After header httprate sets.
//
// The key is the TCP peer (RemoteAddr), which net/http sets and no request
// header can spoof — deliberately chosen over a forwarded-header key so an
// attacker cannot rotate the limit key to bypass throttling. Behind a reverse
// proxy this buckets clients per proxy hop; the edge proxy is expected to apply
// its own per-client limits and this is the backstop.
func PublicRateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return httprate.LimitBy(
		requestsPerMinute,
		time.Minute,
		keyByRemoteIP,
		httprate.WithLimitHandler(rateLimitExceeded),
	)
}

// keyByRemoteIP derives the rate-limit key from the connection peer address,
// normalized so IPv6 clients cannot rotate within their /64 to win fresh
// buckets. It never reads request headers, so the key is unspoofable.
func keyByRemoteIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return httprate.CanonicalizeIP(ip), nil
}

// rateLimitExceeded writes the throttling response in the shared error format.
func rateLimitExceeded(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please retry later")
}
