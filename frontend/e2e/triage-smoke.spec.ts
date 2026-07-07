import { test, expect, DATABASE_URL, withClient, createPerson } from './fixtures';

/**
 * @smoke — triage critical slice (E04, S04.1 / RF-22..RF-26).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. create a person online
 *   2. open the triage form for that person (person_id read from the URL)
 *   3. record the main complaint + one requested service
 *   4. the triage row lands in Postgres, campus-scoped
 *   5. the requested-service junction row is written for the picked service
 *
 * INTENTIONALLY NARROW: campus-boundary, illegal payloads, and offline queueing
 * live in integration tests (testcontainers) — see .project-ai/rules/e2e-test-tiers.md.
 */

test.describe('triage critical slice', () => {
  test('@smoke online create → DB (triage + requested service)', async ({
    page,
    authenticate,
    identity,
  }) => {
    const personName = `${identity.dataPrefix}Ana Triagem`;
    const complaint = `${identity.dataPrefix}dor nas costas`;

    await authenticate();
    const personId = await createPerson(page, personName);

    await page.goto(`/triages/new?person_id=${personId}`);
    await expect(page.getByRole('heading', { name: 'Nova Triagem' })).toBeVisible();

    // Main complaint is the only strictly required field (no htmlFor label →
    // target the textarea by id).
    await page.locator('#triage-main-complaint').fill(complaint);

    // Requested services render as toggle pills (button per service type) inside
    // the group that follows the "Serviços Solicitados" label. Scope to that
    // group so the selector can't match the mobile-nav hamburger or Salvar, then
    // pick the first pill to exercise the junction-table write.
    const serviceGroup = page
      .locator('label', { hasText: 'Serviços Solicitados' })
      .locator('xpath=following-sibling::div[1]');
    const servicePill = serviceGroup.locator('button[type="button"]').first();
    await servicePill.click();

    await page.getByRole('button', { name: 'Salvar Triagem' }).click();
    await expect(page).toHaveURL(/\/triages\/[0-9a-f-]{36}/);

    const row = await withClient(async (client) => {
      const res = await client.query<{
        id: string;
        person_id: string;
        campus_id: string;
      }>(`SELECT id, person_id, campus_id FROM triage WHERE main_complaint = $1`, [complaint]);
      expect(res.rowCount, `triage row should exist in ${DATABASE_URL}`).toBe(1);
      return res.rows[0];
    });
    expect(row?.person_id).toBe(personId);
    expect(row?.campus_id).toBe(identity.campusId);

    // The picked service produced exactly one junction row (cascades on triage
    // delete, so no explicit cleanup needed).
    await withClient(async (client) => {
      const res = await client.query(
        `SELECT service_type_id FROM triage_requested_service WHERE triage_id = $1`,
        [row?.id],
      );
      expect(res.rowCount, 'one requested-service junction row').toBe(1);
    });
  });
});
