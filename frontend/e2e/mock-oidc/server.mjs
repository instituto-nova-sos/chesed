// Mock OIDC issuer for E2E. Stands in for Keycloak.
//
// Why this exists: the Go API validates every request's Bearer token through
// the REAL middleware.OIDCAuth verifier (coreos/go-oidc), which fetches the
// issuer's discovery document and JWKS, then checks the RS256 signature,
// `exp`, `aud`, and `iss`. To exercise that production code path WITHOUT
// running Keycloak, this server publishes a discovery document + JWKS backed
// by an in-process RSA key and mints tokens signed with the matching private
// key. It mirrors backend/internal/integration/auth_middleware_test.go's
// `oidcTestProvider`, but as a standalone service the API and the browser can
// both reach over the Docker network.
//
// Token shape matches docs/20-keycloak-configuration.md: email_verified,
// realm_access.roles, campus_id, person_id.
//
// Endpoints:
//   GET /realms/chesed/.well-known/openid-configuration  → discovery doc
//   GET /realms/chesed/protocol/openid-connect/certs     → JWKS (public key)
//   GET /e2e/token?sub=&email=&roles=&campus_id=&person_id=&exp_seconds=&nonce=
//        → freshly signed token set (JSON: { access_token, id_token,
//          refresh_token, expires_in }). `nonce`, when supplied, is embedded in
//          the id_token so keycloak-js nonce validation passes during the
//          fixture-driven auth-code handshake.
//
// The auth-code/PKCE handshake itself is intercepted in frontend/e2e/fixtures.ts
// (page.route), which captures keycloak-js's nonce and calls /e2e/token to mint
// the matching token set. This keeps the mock a pure signer and the handshake
// logic in the test layer.
//
// SECURITY: this is a TEST-ONLY fixture. The /e2e/token mint endpoint issues
// signed tokens with no authentication. It must NEVER run outside the E2E
// docker-compose network. It is not built into any production image.

import { createServer } from 'node:http';
import { generateKeyPair, exportJWK, SignJWT } from 'jose';

const PORT = Number(process.env.PORT ?? 8080);
// External URL the issuer is reachable at FROM THE TOKEN's point of view.
// Both the API (server-to-server JWKS fetch) and the verifier's issuer check
// resolve against this value, so it must be identical everywhere.
const ISSUER_BASE = process.env.ISSUER_BASE ?? 'http://mock-oidc:8080';
const REALM = process.env.REALM ?? 'chesed';
const CLIENT_ID = process.env.CLIENT_ID ?? 'chesed-pwa';
const KEY_ID = 'e2e-key-1';

const issuer = `${ISSUER_BASE}/realms/${REALM}`;

const { publicKey, privateKey } = await generateKeyPair('RS256');
const publicJwk = await exportJWK(publicKey);
publicJwk.kid = KEY_ID;
publicJwk.use = 'sig';
publicJwk.alg = 'RS256';

const jwks = { keys: [publicJwk] };

const discovery = {
  issuer,
  authorization_endpoint: `${issuer}/protocol/openid-connect/auth`,
  token_endpoint: `${issuer}/protocol/openid-connect/token`,
  jwks_uri: `${issuer}/protocol/openid-connect/certs`,
  userinfo_endpoint: `${issuer}/protocol/openid-connect/userinfo`,
  response_types_supported: ['code'],
  subject_types_supported: ['public'],
  id_token_signing_alg_values_supported: ['RS256'],
};

function sendJSON(res, status, body) {
  const payload = JSON.stringify(body);
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Content-Length': Buffer.byteLength(payload),
    'Cache-Control': 'no-store',
  });
  res.end(payload);
}

async function signJWT(payload, sub, expSeconds, nowSeconds) {
  return new SignJWT(payload)
    .setProtectedHeader({ alg: 'RS256', kid: KEY_ID, typ: 'JWT' })
    .setIssuer(issuer)
    .setAudience(CLIENT_ID)
    .setSubject(sub)
    .setIssuedAt(nowSeconds)
    .setExpirationTime(nowSeconds + expSeconds)
    .sign(privateKey);
}

async function mintToken(params) {
  const nowSeconds = Math.floor(Date.now() / 1000);
  const expSeconds = Number(params.get('exp_seconds') ?? 900);
  const roles = (params.get('roles') ?? 'ADMIN').split(',').filter(Boolean);
  const email = params.get('email') ?? 'e2e@chesed.test';
  const sub = params.get('sub') ?? '11111111-1111-1111-1111-111111111111';
  const nonce = params.get('nonce') ?? undefined;

  const base = {
    email,
    email_verified: (params.get('email_verified') ?? 'true') === 'true',
    realm_access: { roles },
    campus_id: params.get('campus_id') ?? undefined,
    person_id: params.get('person_id') ?? undefined,
    preferred_username: email,
  };

  const accessToken = await signJWT({ ...base, typ: 'Bearer' }, sub, expSeconds, nowSeconds);
  // keycloak-js validates the id_token nonce against the value it generated and
  // stored client-side; embedding it here makes the handshake pass.
  const idToken = await signJWT(
    { ...base, ...(nonce ? { nonce } : {}) },
    sub,
    expSeconds,
    nowSeconds,
  );
  // Refresh token is opaque to the app and never re-validated by our stack; a
  // signed JWT keeps it self-consistent for keycloak-js's bookkeeping.
  const refreshToken = await signJWT({ typ: 'Refresh' }, sub, expSeconds * 4, nowSeconds);

  return {
    access_token: accessToken,
    id_token: idToken,
    refresh_token: refreshToken,
    token_type: 'Bearer',
    expires_in: expSeconds,
    'not-before-policy': 0,
    session_state: '11111111-1111-1111-1111-111111111111',
    scope: 'openid profile email',
  };
}

const server = createServer(async (req, res) => {
  const url = new URL(req.url ?? '/', `http://localhost:${PORT}`);

  if (url.pathname === `/realms/${REALM}/.well-known/openid-configuration`) {
    return sendJSON(res, 200, discovery);
  }
  if (url.pathname === `/realms/${REALM}/protocol/openid-connect/certs`) {
    return sendJSON(res, 200, jwks);
  }
  if (url.pathname === '/e2e/token') {
    try {
      return sendJSON(res, 200, await mintToken(url.searchParams));
    } catch (err) {
      return sendJSON(res, 500, { error: 'mint_failed', message: String(err) });
    }
  }
  if (url.pathname === '/health' || url.pathname === '/') {
    return sendJSON(res, 200, { status: 'ok', issuer });
  }
  return sendJSON(res, 404, { error: 'not_found', path: url.pathname });
});

server.listen(PORT, () => {
  console.log(`[mock-oidc] issuer=${issuer} listening on :${PORT}`);
});
