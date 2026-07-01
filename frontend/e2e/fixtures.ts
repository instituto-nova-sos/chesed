import { test as base, type Page, type Route } from '@playwright/test';
import { Client } from 'pg';
import { randomUUID } from 'node:crypto';

/**
 * E2E fixtures for Chesed.
 *
 * ── Keycloak mock / token injection ───────────────────────────────────────
 * Keycloak is NOT run in the E2E stack. Instead:
 *   1. The `mock-oidc` service (docker-compose.e2e.yml) publishes an OIDC
 *      discovery document + JWKS and mints RS256 tokens signed with a key the
 *      Go API trusts (the API fetches that JWKS). This is the SAME shape the
 *      production middleware.OIDCAuth verifier expects.
 *   2. `authenticate()` below drives the REAL keycloak-js auth-code + PKCE flow
 *      WITHOUT touching any app code. It intercepts only the two OIDC network
 *      hops with `page.route`:
 *        - the `/auth` authorize redirect → 302 back to the app's redirect_uri
 *          with a `code`, echoing keycloak-js's `state` and capturing its
 *          `nonce`;
 *        - the `/token` exchange → a token set minted by the mock issuer with
 *          the captured `nonce` (so keycloak-js nonce validation passes) and the
 *          per-test claims (email_verified=true, a realm role, campus_id).
 *      keycloak-js then stores the token and the app boots authenticated — no
 *      Keycloak login page, no app changes.
 *
 * The injected token carries: email_verified=true, realm_access.roles, and a
 * campus_id matching the seeded test campus — matching docs/20-keycloak-
 * configuration.md's documented claim set.
 *
 * ── Per-test campus isolation + cleanup ───────────────────────────────────
 * The Go API resolves campus_id from the app_user row (NOT the JWT): the
 * AutoProvision middleware looks up app_user by keycloak_subject_id and rejects
 * with 403 when no campus is assigned. So a token alone is not enough — each
 * test must have a provisioned app_user.
 *
 * Strategy:
 *   - Each test gets a UNIQUE keycloak subject (uuid) and a UNIQUE data prefix.
 *   - `seedAppUser()` inserts an app_user (subject, ADMIN, campus = the default
 *     campus seeded by migration 000011) directly into Postgres before the test.
 *     ADMIN is used so the OnboardingGuard's profile/campus completion steps are
 *     bypassed (it still respects email verification + agreement, which ADMIN
 *     without a VOLUNTEER role does not require — see middleware.RequireAgreement).
 *   - `afterEach` deletes every person whose full_name starts with the test's
 *     unique prefix, then the seeded app_user. This keeps tests independent and
 *     leaves the DB clean.
 *
 * NO dedicated cleanup API endpoint exists in the backend, so teardown talks to
 * Postgres directly (the E2E stack publishes it on port 5433) AND uses a
 * per-run unique name prefix to scope deletes. If a future endpoint is added,
 * swap the SQL deletes for API calls here.
 */

const OIDC_TOKEN_URL = process.env.E2E_OIDC_TOKEN_URL ?? 'http://localhost:8181/e2e/token';
const DATABASE_URL =
  process.env.E2E_DATABASE_URL ?? 'postgres://chesed:chesed@localhost:5433/chesed?sslmode=disable';

// The default campus inserted by migration 000011_seed_data.up.sql. Reused as
// the deterministic test campus so every test is scoped to a known campus_id.
const TEST_CAMPUS_ID = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11';

export interface E2EIdentity {
  subject: string;
  email: string;
  roles: string[];
  campusId: string;
  /** Unique per-test prefix for any data this test creates (cleanup scope). */
  dataPrefix: string;
}

interface E2EFixtures {
  identity: E2EIdentity;
  authenticate: () => Promise<void>;
}

async function withClient<T>(fn: (client: Client) => Promise<T>): Promise<T> {
  const client = new Client({ connectionString: DATABASE_URL });
  await client.connect();
  try {
    return await fn(client);
  } finally {
    await client.end();
  }
}

async function seedAppUser(identity: E2EIdentity): Promise<void> {
  await withClient(async (client) => {
    await client.query(
      `INSERT INTO app_user (id, email, keycloak_subject_id, access_profile, campus_id, is_active)
       VALUES ($1, $2, $3, 'ADMIN', $4, TRUE)
       ON CONFLICT (keycloak_subject_id) DO UPDATE
         SET campus_id = EXCLUDED.campus_id, is_active = TRUE`,
      [randomUUID(), identity.email, identity.subject, identity.campusId],
    );
  });
}

async function cleanupTestData(identity: E2EIdentity): Promise<void> {
  await withClient(async (client) => {
    // Persons created by this test (scoped by the unique name prefix).
    await client.query(`DELETE FROM person WHERE full_name LIKE $1`, [`${identity.dataPrefix}%`]);
    await client.query(`DELETE FROM app_user WHERE keycloak_subject_id = $1`, [identity.subject]);
  });
}

/**
 * Intercepts the two OIDC hops keycloak-js performs and feeds it a token set
 * minted by the mock issuer. Registered on the page BEFORE navigation.
 */
async function installOidcInterception(page: Page, identity: E2EIdentity): Promise<void> {
  // Capture the nonce keycloak-js puts on the authorize request so the minted
  // id_token carries the same value (keycloak-js rejects a mismatched nonce).
  let capturedNonce = '';

  // /auth authorize endpoint → 302 to the app callback with code + state.
  await page.route('**/realms/*/protocol/openid-connect/auth*', async (route: Route) => {
    const url = new URL(route.request().url());
    const state = url.searchParams.get('state') ?? '';
    capturedNonce = url.searchParams.get('nonce') ?? '';
    const redirectUri = url.searchParams.get('redirect_uri') ?? '';
    const callback = `${redirectUri}#state=${encodeURIComponent(state)}&session_state=e2e&code=E2E_AUTH_CODE`;
    await route.fulfill({ status: 302, headers: { location: callback }, body: '' });
  });

  // /token exchange → mint a token set with the captured nonce + test claims.
  await page.route('**/realms/*/protocol/openid-connect/token', async (route: Route) => {
    const params = new URLSearchParams({
      sub: identity.subject,
      email: identity.email,
      roles: identity.roles.join(','),
      campus_id: identity.campusId,
      nonce: capturedNonce,
    });
    const res = await fetch(`${OIDC_TOKEN_URL}?${params.toString()}`);
    const body = await res.text();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json', 'cache-control': 'no-store' },
      body,
    });
  });
}

export const test = base.extend<E2EFixtures>({
  identity: async ({}, use, testInfo) => {
    const unique = randomUUID().slice(0, 8);
    const identity: E2EIdentity = {
      subject: randomUUID(),
      email: `e2e-${unique}@chesed.test`,
      roles: ['ADMIN'],
      campusId: TEST_CAMPUS_ID,
      // Title-derived + unique so concurrent/leftover data never collides.
      dataPrefix: `E2E_${testInfo.title.replace(/[^A-Za-z0-9]/g, '_').slice(0, 20)}_${unique}_`,
    };

    await seedAppUser(identity);
    await use(identity);
    await cleanupTestData(identity);
  },

  authenticate: async ({ page, identity }, use) => {
    const run = async () => {
      await installOidcInterception(page, identity);
      // Navigate; keycloak-js (onLoad: 'login-required') triggers the handshake.
      // Our routes complete it (authorize → callback → token exchange) and the
      // app boots authenticated. waitUntil:'networkidle' lets the multi-hop
      // redirect + token POST settle before the test proceeds.
      await page.goto('/', { waitUntil: 'networkidle' });
    };
    await use(run);
  },
});

export { expect } from '@playwright/test';
export { TEST_CAMPUS_ID, DATABASE_URL, withClient };
