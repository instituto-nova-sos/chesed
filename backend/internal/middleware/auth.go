package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
)

type keycloakClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	RealmAccess   struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	PersonID string `json:"person_id"`
}

// OIDCAuth creates middleware that validates Keycloak-issued JWT tokens.
// It fetches JWKS from the discovery endpoint and caches keys automatically.
// discoveryURL is used to fetch OIDC configuration (e.g., http://keycloak:8080/realms/chesed inside Docker).
// The actual issuer in tokens may differ (e.g., http://localhost:8180/realms/chesed) when
// Keycloak's KC_HOSTNAME is configured for external access. The provider resolves
// the canonical issuer from the discovery document and validates tokens against it.
func OIDCAuth(discoveryURL, clientID string, skipIssuerCheck bool) (func(http.Handler) http.Handler, error) {
	provider, err := newProviderWithRetry(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("middleware.OIDCAuth: %w", err)
	}

	if skipIssuerCheck {
		slog.Warn("middleware.OIDCAuth: SkipIssuerCheck is enabled — disable in production",
			"discoveryURL", discoveryURL)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID:        clientID,
		SkipIssuerCheck: skipIssuerCheck,
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, authErr := authenticateRequest(r, verifier)
			if authErr != nil {
				writeError(w, authErr.status, authErr.code, authErr.message)
				return
			}
			ctx := auth.NewContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// authError carries the HTTP response for a failed authentication.
type authError struct {
	status  int
	code    string
	message string
}

// authenticateRequest extracts and verifies the bearer token, validates the
// claims, and enforces the email-verified gate.
func authenticateRequest(r *http.Request, verifier *oidc.IDTokenVerifier) (auth.AuthClaims, *authError) {
	token, err := extractBearerToken(r)
	if err != nil {
		return auth.AuthClaims{}, &authError{http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header"}
	}

	idToken, err := verifier.Verify(r.Context(), token)
	if err != nil {
		slog.WarnContext(r.Context(), "middleware.OIDCAuth: token verification failed", "error", err.Error())
		return auth.AuthClaims{}, &authError{http.StatusUnauthorized, "unauthorized", "invalid or expired token"}
	}

	claims, err := extractClaims(idToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "middleware.OIDCAuth: claims extraction failed", "error", err.Error())
		return auth.AuthClaims{}, &authError{http.StatusUnauthorized, "unauthorized", "invalid token claims"}
	}

	if !claims.EmailVerified {
		slog.WarnContext(r.Context(), "middleware.OIDCAuth: email not verified",
			"subject", claims.Subject, "email", claims.Email)
		return auth.AuthClaims{}, &authError{http.StatusForbidden, "forbidden", "email not verified"}
	}

	return claims, nil
}

func newProviderWithRetry(issuerURL string) (*oidc.Provider, error) {
	// InsecureIssuerURLContext allows the provider to fetch OIDC discovery
	// from a different URL than the token issuer. Required in Docker where
	// the API reaches Keycloak at http://keycloak:8080 but tokens have
	// issuer http://localhost:8180 (the externally-accessible URL).
	ctx := oidc.InsecureIssuerURLContext(context.Background(), issuerURL)
	maxRetries := 15
	retryInterval := 2 * time.Second

	var lastErr error
	for i := range maxRetries {
		provider, err := oidc.NewProvider(ctx, issuerURL)
		if err == nil {
			return provider, nil
		}
		lastErr = err
		slog.Info("middleware.OIDCAuth: waiting for OIDC provider",
			"attempt", i+1, "max", maxRetries,
		)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to connect to OIDC provider after %d attempts: %w", maxRetries, lastErr)
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	return parts[1], nil
}

func extractClaims(idToken *oidc.IDToken) (auth.AuthClaims, error) {
	var kc keycloakClaims
	if err := idToken.Claims(&kc); err != nil {
		return auth.AuthClaims{}, fmt.Errorf("extractClaims: %w", err)
	}

	var personID uuid.UUID
	if kc.PersonID != "" {
		parsed, err := uuid.Parse(kc.PersonID)
		if err != nil {
			return auth.AuthClaims{}, fmt.Errorf("extractClaims: invalid person_id: %w", err)
		}
		personID = parsed
	}

	return auth.AuthClaims{
		Subject:       idToken.Subject,
		Email:         kc.Email,
		EmailVerified: kc.EmailVerified,
		Roles:         kc.RealmAccess.Roles,
		PersonID:      personID,
	}, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]string{
		"error":   code,
		"message": message,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("middleware.writeError: failed to encode response", "error", err)
	}
}
