import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for Chesed E2E.
 *
 * Topology (see docs/e2e-testing.md):
 *   - The Go API + Postgres + mock OIDC issuer run via docker-compose.e2e.yml
 *     (operator brings these up BEFORE running Playwright).
 *   - This config serves the BUILT frontend with `vite preview` on port 5173.
 *
 * Port 5173 is NOT arbitrary: the API's CORS allow-list is hardcoded to
 * `http://localhost:5173` (backend main.go), so the preview MUST serve there
 * or every cross-origin API call is blocked. The built bundle is pointed at the
 * E2E API (port 8081) via VITE_API_BASE_URL, injected at build time below.
 *
 * Tag convention: smoke specs are tagged `@smoke` in their title; the
 * `test:e2e:smoke` npm script runs `--grep @smoke`.
 */

const PREVIEW_PORT = 5173;
const BASE_URL = `http://localhost:${PREVIEW_PORT}`;

// Host-facing URLs for the dockerized E2E stack (port offsets from the dev
// stack so both can run side by side).
const E2E_API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'http://localhost:8081/api/v1';
const E2E_OIDC_TOKEN_URL =
  process.env.E2E_OIDC_TOKEN_URL ?? 'http://localhost:8181/e2e/token';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.ts',
  // Generous CI-less local timeouts: the offline→online sync poll needs slack.
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never' }]],

  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  // Discoverability only. The fixtures read process.env (E2E_API_BASE_URL,
  // E2E_OIDC_TOKEN_URL, E2E_DATABASE_URL) directly so they work outside the
  // Playwright runner too; these mirror the defaults for documentation.
  metadata: {
    apiBaseUrl: E2E_API_BASE_URL,
    oidcTokenUrl: E2E_OIDC_TOKEN_URL,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Build then preview the frontend, pointed at the E2E API. The build runs
  // every invocation so the bundle always matches the current source; this is
  // intentional for an honest E2E (no stale dist).
  webServer: {
    command: `npm run build && npm run preview -- --port ${PREVIEW_PORT} --strictPort`,
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: {
      VITE_API_BASE_URL: E2E_API_BASE_URL,
      // The frontend never talks to the mock issuer directly (the fixture
      // injects the token), but keep these defined so keycloak-js construction
      // does not choke on undefined envs.
      VITE_KEYCLOAK_URL: 'http://localhost:8181',
      VITE_KEYCLOAK_REALM: 'chesed',
      VITE_KEYCLOAK_CLIENT_ID: 'chesed-pwa',
    },
  },
});
