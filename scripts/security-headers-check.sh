#!/usr/bin/env bash
#
# security-headers-check.sh
#
# Asserts that every HTTP response from the Chesed API carries the security
# headers set by backend/internal/middleware/security_headers.go, with their
# exact expected values. Strict-Transport-Security (HSTS) is HTTPS-only and
# flag-gated (SecurityHeadersWith(hstsEnabled) behind HSTS_ENABLED), so it is
# asserted ONLY when the target URL uses https.
#
# Usage:
#   scripts/security-headers-check.sh [URL]
#   SECURITY_CHECK_URL=https://api.example.com/api/v1/health scripts/security-headers-check.sh
#
# The URL defaults to $SECURITY_CHECK_URL, then to http://localhost:8080/api/v1/health.
# Exit code 0 when all required headers are present and correct; non-zero (1)
# when any header is missing or has an unexpected value. Curl/transport failures
# exit 2. CI-usable: no interactive prompts, deterministic output.

set -euo pipefail

DEFAULT_URL="http://localhost:8080/api/v1/health"
TARGET_URL="${1:-${SECURITY_CHECK_URL:-$DEFAULT_URL}}"

FAILED=0

# Fetch response headers once; fail fast on a transport-level error so a down
# server is not misreported as "all headers missing".
HEADERS_FILE="$(mktemp)"
trap 'rm -f "$HEADERS_FILE"' EXIT

if ! curl -sS -o /dev/null -D "$HEADERS_FILE" "$TARGET_URL"; then
    echo "ERROR: could not reach ${TARGET_URL}" >&2
    exit 2
fi

# header_value <name> -> echoes the (last) value for a header, CR-stripped.
header_value() {
    local name="$1"
    # -i case-insensitive; take the last occurrence; strip "Name:" prefix and CR.
    grep -i "^${name}:" "$HEADERS_FILE" \
        | tail -n1 \
        | sed -E "s/^[^:]+:[[:space:]]*//I" \
        | tr -d '\r'
}

# assert_header <name> <expected-exact-value>
assert_header() {
    local name="$1"
    local expected="$2"
    local actual
    actual="$(header_value "$name")"

    if [[ -z "$actual" ]]; then
        echo "FAIL: missing header '${name}' (expected '${expected}')"
        FAILED=1
    elif [[ "$actual" != "$expected" ]]; then
        echo "FAIL: header '${name}' expected '${expected}', got '${actual}'"
        FAILED=1
    else
        echo "PASS: ${name}: ${actual}"
    fi
}

echo "Checking security headers on ${TARGET_URL}"
echo "---"

# Values mirror backend/internal/middleware/security_headers.go exactly.
assert_header "X-Content-Type-Options" "nosniff"
assert_header "X-Frame-Options" "DENY"
assert_header "Referrer-Policy" "strict-origin-when-cross-origin"
assert_header "Permissions-Policy" "camera=(), microphone=(), geolocation=()"
assert_header "Content-Security-Policy" "default-src 'none'; frame-ancestors 'none'"
assert_header "Cache-Control" "no-store"

# HSTS is HTTPS-only and gated behind HSTS_ENABLED via SecurityHeadersWith.
# Only assert it when the target is served over TLS; asserting it on a plain
# http:// target would be a false negative (HSTS over cleartext is meaningless
# and browsers ignore it).
if [[ "$TARGET_URL" == https://* ]]; then
    assert_header "Strict-Transport-Security" "max-age=63072000; includeSubDomains; preload"
else
    echo "SKIP: Strict-Transport-Security (target is not https; HSTS is HTTPS-only)"
fi

echo "---"
if [[ "$FAILED" -ne 0 ]]; then
    echo "Security headers check FAILED"
    exit 1
fi

echo "Security headers check PASSED"
