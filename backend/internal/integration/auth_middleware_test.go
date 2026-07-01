//go:build integration

// This file is the COMPENSATING coverage for the real auth path.
//
// The rest of the integration suite (see harness_test.go) deliberately
// BYPASSES authentication: it injects a synthetic auth.AuthClaims directly
// into the request context so application-logic tests stay focused on
// service/repository behaviour. That leaves the real Keycloak token-validation
// middleware (middleware.OIDCAuth) untested — the seam where production data
// crosses the network boundary from an external identity provider.
//
// TestAuthMiddleware closes that gap. It drives requests through the REAL
// middleware.OIDCAuth verifier (coreos/go-oidc v3) against a LOCAL OIDC
// provider stood up with httptest: the test serves its own discovery document
// and JWKS (public half of an RSA key generated in-process) and mints signed
// tokens with the matching private key using go-jose v4. No Docker, no
// Postgres, no Keycloak — but the exact production verification code runs.
//
// We do NOT wire the full chi → service → repository stack here. The thing
// under test is the token-validation middleware itself, so we wrap a trivial
// 200 handler with the real middleware. This keeps the assertion surface on
// the auth contract: valid → pass, expired → 401, unverified email → 403,
// bad signature → 401. (CLAUDE.md §9: OIDC via coreos/go-oidc; §12: the API
// MUST check email_verified and reject unverified users with HTTP 403.)
package integration_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
)

const (
	authTestClientID = "chesed-test-client"
	authTestKeyID    = "test-key-1"
)

// oidcTestProvider is a local stand-in for Keycloak. It serves the OIDC
// discovery document and a JWKS containing the public half of testKey, and
// mints signed tokens with the private half so the REAL verifier accepts them.
type oidcTestProvider struct {
	server   *httptest.Server
	signer   jose.Signer
	otherKey *rsa.PrivateKey // a second, unpublished key used to forge bad signatures
}

// newOIDCTestProvider boots the local provider. The issuer in tokens equals the
// httptest server URL, matching the discovery document's "issuer" field, so the
// verifier's issuer check passes without SkipIssuerCheck.
func newOIDCTestProvider(t *testing.T) *oidcTestProvider {
	t.Helper()

	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate signing key")

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "generate forging key")

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: signingKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", authTestKeyID),
	)
	require.NoError(t, err, "build jose signer")

	jwks := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       signingKey.Public(),
				KeyID:     authTestKeyID,
				Algorithm: string(jose.RS256),
				Use:       "sig",
			},
		},
	}

	p := &oidcTestProvider{signer: signer, otherKey: otherKey}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		doc := map[string]any{
			"issuer":                                p.issuer(),
			"authorization_endpoint":                p.issuer() + "/auth",
			"token_endpoint":                        p.issuer() + "/token",
			"jwks_uri":                              p.issuer() + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		writeJSON(w, doc)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, jwks)
	})

	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	return p
}

func (p *oidcTestProvider) issuer() string { return p.server.URL }

// tokenClaims is the minimal Keycloak-shaped claim set the middleware reads.
type tokenClaims struct {
	Issuer        string `json:"iss"`
	Subject       string `json:"sub"`
	Audience      string `json:"aud"`
	Expiry        int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	RealmAccess   struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// mint signs a token with the published key (valid signature path).
func (p *oidcTestProvider) mint(t *testing.T, c tokenClaims) string {
	t.Helper()
	c.Issuer = p.issuer()
	c.Audience = authTestClientID
	payload, err := json.Marshal(c)
	require.NoError(t, err, "marshal claims")

	obj, err := p.signer.Sign(payload)
	require.NoError(t, err, "sign token")

	serialized, err := obj.CompactSerialize()
	require.NoError(t, err, "serialize token")
	return serialized
}

// mintWithBadSignature signs with a key that is NOT in the served JWKS, so the
// verifier rejects the signature even though every claim is otherwise valid.
func (p *oidcTestProvider) mintWithBadSignature(t *testing.T, c tokenClaims) string {
	t.Helper()
	c.Issuer = p.issuer()
	c.Audience = authTestClientID
	payload, err := json.Marshal(c)
	require.NoError(t, err, "marshal claims")

	forger, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: p.otherKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", authTestKeyID),
	)
	require.NoError(t, err, "build forging signer")

	obj, err := forger.Sign(payload)
	require.NoError(t, err, "sign forged token")

	serialized, err := obj.CompactSerialize()
	require.NoError(t, err, "serialize forged token")
	return serialized
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// validClaims returns a claim set that should be accepted: future expiry,
// verified email, a role, and the matching audience/issuer (set at mint time).
func validClaims() tokenClaims {
	now := time.Now()
	c := tokenClaims{
		Subject:       "11111111-1111-1111-1111-111111111111",
		Expiry:        now.Add(15 * time.Minute).Unix(),
		IssuedAt:      now.Unix(),
		Email:         "verified@chesed.test",
		EmailVerified: true,
	}
	c.RealmAccess.Roles = []string{"VOLUNTEER"}
	return c
}

// TestAuthMiddleware exercises the real middleware.OIDCAuth verifier against a
// local OIDC provider. Each sub-case mints a token, sends it through the real
// middleware wrapping a trivial 200 handler, and asserts the HTTP outcome plus
// (on success) that validated claims reach the downstream handler context.
func TestAuthMiddleware(t *testing.T) {
	provider := newOIDCTestProvider(t)

	authMW, err := middleware.OIDCAuth(provider.issuer(), authTestClientID, false)
	require.NoError(t, err, "build real OIDC middleware against local provider")

	// downstream captures the claims the middleware injected, so the happy path
	// can assert the full validate → context propagation contract, not just 200.
	var gotClaims auth.AuthClaims
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = auth.ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := authMW(downstream)

	tests := []struct {
		name       string
		token      func(t *testing.T) string
		omitHeader bool
		wantStatus int
		wantErrKey string
	}{
		{
			name:       "valid token passes",
			token:      func(t *testing.T) string { return provider.mint(t, validClaims()) },
			wantStatus: http.StatusOK,
		},
		{
			name: "expired token is rejected with 401",
			token: func(t *testing.T) string {
				c := validClaims()
				past := time.Now().Add(-1 * time.Hour)
				c.IssuedAt = past.Add(-15 * time.Minute).Unix()
				c.Expiry = past.Unix()
				return provider.mint(t, c)
			},
			wantStatus: http.StatusUnauthorized,
			wantErrKey: "unauthorized",
		},
		{
			name: "unverified email is rejected with 403",
			token: func(t *testing.T) string {
				c := validClaims()
				c.EmailVerified = false
				return provider.mint(t, c)
			},
			wantStatus: http.StatusForbidden,
			wantErrKey: "forbidden",
		},
		{
			name: "invalid signature is rejected with 401",
			token: func(t *testing.T) string {
				return provider.mintWithBadSignature(t, validClaims())
			},
			wantStatus: http.StatusUnauthorized,
			wantErrKey: "unauthorized",
		},
		{
			name:       "missing authorization header is rejected with 401",
			omitHeader: true,
			wantStatus: http.StatusUnauthorized,
			wantErrKey: "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClaims = auth.AuthClaims{}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			req = req.WithContext(context.Background())
			if !tt.omitHeader {
				req.Header.Set("Authorization", "Bearer "+tt.token(t))
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, "unexpected status; body=%s", rec.Body.String())

			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, "verified@chesed.test", gotClaims.Email,
					"validated claims must reach the downstream handler")
				assert.True(t, gotClaims.EmailVerified)
				assert.Contains(t, gotClaims.Roles, "VOLUNTEER")
				return
			}

			// Error responses must carry the documented JSON error envelope and
			// must NOT have leaked into the protected handler.
			assert.Equal(t, auth.AuthClaims{}, gotClaims,
				"rejected request must never reach the downstream handler")

			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body must be JSON")
			assert.Equal(t, tt.wantErrKey, body["error"])
			assert.NotEmpty(t, body["message"])
		})
	}
}
