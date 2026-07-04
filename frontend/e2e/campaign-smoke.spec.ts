import { test, expect, DATABASE_URL, withClient } from './fixtures';

/**
 * @smoke — narrow critical slice of the campaign flow (S07.6 follow-up).
 *
 * What this proves end-to-end (screen → real Go API → real Postgres):
 *   1. login as ADMIN (token injected, Keycloak mocked)
 *   2. create one campaign while ONLINE via the form
 *   3. the campaign is visible in the list (served from the API)
 *   4. the row really exists in Postgres, campus-scoped, with PLANNED status
 *
 * INTENTIONALLY NARROW: team management, metrics, status filters, and the
 * campaign link on triage/attendance live in integration tests (MSW +
 * testcontainers), not here. See .project-ai/rules/e2e-test-tiers.md.
 */

test.describe('campaign critical slice', () => {
  test('@smoke online create → list → DB', async ({ page, authenticate, identity }) => {
    const campaignName = `${identity.dataPrefix}Campanha Inverno`;

    await authenticate();

    // The role-gated "Nova campanha" link rendering confirms the ADMIN
    // identity settled past every auth/onboarding guard.
    await page.goto('/campaigns');
    await page.getByRole('link', { name: 'Nova campanha' }).click();
    await expect(page).toHaveURL(/\/campaigns\/new/);

    // Fill the required fields (type defaults to SOCIAL_ACTION) and submit.
    await page.getByLabel('Nome').fill(campaignName);
    await page.getByLabel('Data de início').fill('2026-07-10');
    await page.getByRole('button', { name: 'Salvar' }).click();

    // On success the app navigates to the detail page and shows the name.
    await expect(page).toHaveURL(/\/campaigns\/[0-9a-f-]{36}/);
    await expect(page.getByRole('heading', { name: campaignName })).toBeVisible();

    // The campaign is visible in the list (served from the API).
    await page.goto('/campaigns');
    await expect(page.getByText(campaignName)).toBeVisible();

    // DB-level assertion: the row landed in Postgres on the test campus with
    // the documented PLANNED default status.
    const row = await withClient(async (client) => {
      const res = await client.query<{ campus_id: string; status: string }>(
        `SELECT campus_id, status FROM campaign WHERE name = $1`,
        [campaignName],
      );
      expect(res.rowCount, `campaign row should exist in ${DATABASE_URL}`).toBe(1);
      return res.rows[0];
    });
    expect(row?.campus_id).toBe(identity.campusId);
    expect(row?.status).toBe('PLANNED');
  });
});
