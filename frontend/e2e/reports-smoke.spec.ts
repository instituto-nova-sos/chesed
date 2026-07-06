import { test, expect } from './fixtures';

/**
 * @smoke — narrow critical slice of the reports & dashboard flow (E10,
 * S10.2/S10.3).
 *
 * What this proves end-to-end (screen → real Go API → real Postgres):
 *   1. login as ADMIN (token injected, Keycloak mocked)
 *   2. the dashboard KPIs render from GET /reports/dashboard (server-computed,
 *      campus-scoped) — not recomputed client-side
 *   3. a chart section renders on the dashboard
 *   4. the reports page generates an attendance report and renders its charts
 *
 * INTENTIONALLY NARROW: the filter contract, by_professional aggregation,
 * campus boundary, and Coordinator+ RBAC live in integration tests (MSW +
 * testcontainers), not here. See .project-ai/rules/e2e-test-tiers.md. This
 * slice is read-only, so it seeds no rows and needs no cleanup.
 */

test.describe('reports & dashboard critical slice', () => {
  test('@smoke dashboard KPIs + report charts render from the API', async ({
    page,
    authenticate,
  }) => {
    await authenticate();

    // ADMIN lands on the dashboard (index route). The KPI card renders only
    // after GET /reports/dashboard resolves — its presence confirms the
    // server endpoint answered past every auth/onboarding guard.
    await expect(
      page.getByRole('heading', { name: 'Dashboard' }),
    ).toBeVisible();
    await expect(page.getByText('Pessoas cadastradas')).toBeVisible();
    await expect(
      page.getByRole('heading', { name: 'Atendimentos por mês' }),
    ).toBeVisible();

    // The reports page: generate a report for the default period and confirm
    // the chart-backed breakdown renders (served from GET /reports/attendances).
    await page.goto('/reports');
    await expect(
      page.getByRole('heading', { name: 'Relatórios' }),
    ).toBeVisible();
    await page.getByRole('button', { name: 'Gerar relatório' }).click();

    // The status-chart card header is a stable, chart-specific marker that
    // only renders once a report has been generated.
    await expect(
      page.getByRole('heading', { name: 'Atendimentos por status' }),
    ).toBeVisible();
  });
});
