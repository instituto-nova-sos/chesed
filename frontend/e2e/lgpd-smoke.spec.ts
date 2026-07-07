import { test, expect } from './fixtures';

/**
 * @smoke — LGPD compliance surfaces critical slice (E11, S11.4/S11.6 / RF-53, RNF-01).
 *
 * The audit viewer and the compliance report are ADMIN/COORDINATOR read-only
 * surfaces over existing data. This slice proves both render from the real API
 * end-to-end (screen → Go API → Postgres): the audit log auto-queries a default
 * range on mount and the compliance report generates campus-scoped metrics.
 *
 * INTENTIONALLY NARROW: filter permutations, CSV export bytes, and RBAC denials
 * live in integration tests. audit_log is append-only, so nothing is created or
 * cleaned up here.
 */

test.describe('LGPD compliance surfaces critical slice', () => {
  test('@smoke audit viewer renders the log table from the API', async ({
    page,
    authenticate,
  }) => {
    await authenticate();

    await page.goto('/audit-logs');
    await expect(page.getByRole('heading', { name: 'Registros de auditoria' })).toBeVisible();

    // The default range auto-queries on mount; the table (or its empty state)
    // must render. Assert the column header is present — this only renders once
    // the API responded with the table shape.
    await expect(page.getByRole('columnheader', { name: 'Ação' })).toBeVisible();
  });

  test('@smoke compliance report generates campus-scoped metrics', async ({
    page,
    authenticate,
  }) => {
    await authenticate();

    await page.goto('/compliance');
    await expect(page.getByRole('heading', { name: 'Conformidade (LGPD)' })).toBeVisible();

    // Fill the required date range and generate. A wide window guarantees the
    // aggregate query runs against seeded/prior data.
    await page.getByLabel('Início').fill('2020-01-01');
    await page.getByLabel('Fim').fill('2030-12-31');
    await page.getByRole('button', { name: 'Gerar relatório' }).click();

    // The report renders its metric cards from the API response.
    await expect(page.getByText('Consentimentos ativos')).toBeVisible();
    await expect(page.getByText('Titulares de dados')).toBeVisible();

    // The CSV export becomes enabled once a report exists (S11.4 EXPORT path).
    await expect(page.getByRole('button', { name: 'Exportar CSV' })).toBeEnabled();
  });
});
