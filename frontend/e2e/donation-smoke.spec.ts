import { test, expect, DATABASE_URL, withClient } from './fixtures';

/**
 * @smoke — narrow critical slice of the donation flow (E09, S09.1).
 *
 * What this proves end-to-end (screen → real Go API → real Postgres):
 *   1. login as ADMIN (token injected, Keycloak mocked)
 *   2. create one FINANCIAL donation while ONLINE via the form
 *   3. the donation is visible in the list (served from the API)
 *   4. the row really exists in Postgres, campus-scoped, with the amount
 *
 * INTENTIONALLY NARROW: campaign/donor linkage, edit, GOODS/SERVICES
 * validation, and RBAC boundaries live in integration tests (MSW +
 * testcontainers), not here. See .project-ai/rules/e2e-test-tiers.md.
 */

test.describe('donation critical slice', () => {
  test('@smoke online create → list → DB', async ({ page, authenticate, identity }) => {
    const notes = `${identity.dataPrefix}Doação Inverno`;

    await authenticate();

    // The role-gated "Nova doação" link rendering confirms the ADMIN
    // identity settled past every auth/onboarding guard.
    await page.goto('/donations');
    await page.getByRole('link', { name: 'Nova doação' }).click();
    await expect(page).toHaveURL(/\/donations\/new/);

    // Type defaults to FINANCIAL, so the amount field is visible. Fill the
    // required amount and a distinctive note. Sprint 9.3: pick a non-default
    // currency (USD) from the select to prove multi-currency end-to-end.
    await page.getByLabel('Valor').fill('175.50');
    await page.getByLabel('Moeda').selectOption('USD');
    await page.getByLabel('Observações').fill(notes);
    await page.getByRole('button', { name: 'Salvar' }).click();

    // On success the app navigates to the detail page.
    await expect(page).toHaveURL(/\/donations\/[0-9a-f-]{36}/);
    await expect(page.getByRole('heading', { name: 'Doação' })).toBeVisible();

    // The donation is visible in the list (served from the API). Assert on the
    // formatted amount cell — unique to this row and rendered only in the table
    // (a plain `Financeira` text match would collide with the type-filter's
    // hidden <option> elements).
    await page.goto('/donations');
    await expect(page.getByRole('cell', { name: /175,50/ })).toBeVisible();

    // DB-level assertion: the row landed in Postgres on the test campus with
    // the submitted amount and FINANCIAL type.
    const row = await withClient(async (client) => {
      const res = await client.query<{
        campus_id: string;
        donation_type: string;
        amount: string;
        currency: string;
      }>(
        `SELECT campus_id, donation_type, amount, currency FROM donation WHERE notes = $1`,
        [notes],
      );
      expect(res.rowCount, `donation row should exist in ${DATABASE_URL}`).toBe(1);
      return res.rows[0];
    });
    expect(row?.campus_id).toBe(identity.campusId);
    expect(row?.donation_type).toBe('FINANCIAL');
    expect(Number(row?.amount)).toBe(175.5);
    expect(row?.currency).toBe('USD');
  });
});
