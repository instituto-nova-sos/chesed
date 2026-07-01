import { test, expect, DATABASE_URL, withClient } from './fixtures';

/**
 * @smoke — narrow critical slice of the offline-first person flow.
 *
 * What this proves end-to-end (screen → real Go API → real Postgres):
 *   1. login (token injected, Keycloak mocked)
 *   2. create one person while ONLINE
 *   3. the person is visible in the list (served from the API)
 *   4. the row really exists in Postgres (DB-level assertion)
 *
 * INTENTIONALLY NARROW: pull, conflict resolution, batching, and reports live in
 * integration tests (MSW + testcontainers), not here. See
 * .project-ai/rules/e2e-test-tiers.md.
 *
 * OFFLINE PORTION — ACTIVE. The sync drainer is wired into the running app
 * (useOnlineSync in AppLayout auto-drains on reconnect; usePersons/useTriages
 * serve the Dexie cache offline). Each offline slice below creates a record with
 * the browser offline, confirms it renders from the IndexedDB cache, reconnects,
 * and asserts the drainer flushed the syncQueue to /api/v1/sync/push and the row
 * reached Postgres. Person covers the base case; triage covers the S05.2
 * per-entity offline-create path against the real stack.
 */

test.describe('person offline-first critical slice', () => {
  test('@smoke online create → list → DB', async ({ page, authenticate, identity }) => {
    const personName = `${identity.dataPrefix}Maria Silva`;

    await authenticate();

    // Go to the person list (only reachable inside the authenticated AppLayout)
    // and open the create form. The "Nova Pessoa" button rendering confirms the
    // handshake settled and we are past every auth/onboarding guard.
    await page.goto('/persons');
    await page.getByRole('button', { name: 'Nova Pessoa' }).click();
    await expect(page).toHaveURL(/\/persons\/new/);

    // Fill the only required field and submit (document is optional; leaving CPF
    // blank avoids the CPF-format refinement).
    // The name <input> has no htmlFor-associated label, so target it by its
    // placeholder (the only stable, unique handle on the create form).
    await page.getByPlaceholder('Ex: Maria da Silva Santos').fill(personName);
    await page.getByRole('button', { name: 'Cadastrar Pessoa' }).click();

    // On success the app navigates to the detail page and shows the name.
    await expect(page).toHaveURL(/\/persons\/[0-9a-f-]{36}/);
    await expect(page.getByRole('heading', { name: personName })).toBeVisible();

    // The person is visible in the list (served from the API).
    await page.goto('/persons');
    await expect(page.getByText(personName)).toBeVisible();

    // DB-level assertion: the row really landed in Postgres on the test campus.
    const dbName = await withClient(async (client) => {
      const res = await client.query<{ full_name: string; campus_id: string }>(
        `SELECT full_name, campus_id FROM person WHERE full_name = $1`,
        [personName],
      );
      expect(res.rowCount, `person row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.campus_id).toBe(identity.campusId);
      return res.rows[0]?.full_name;
    });
    expect(dbName).toBe(personName);
  });

  // ── OFFLINE SLICE — active now that the sync drainer ships ─────────────────
  // Offline create persists to Dexie + the sync queue; the list is served from
  // the IndexedDB cache while offline; the drainer flushes to /sync/push on
  // reconnect (useOnlineSync auto-drains on the browser 'online' event).
  test(
    '@smoke offline person stays visible then queue drains on reconnect',
    async ({ page, authenticate, context, identity }) => {
      const personName = `${identity.dataPrefix}Offline Joao`;

      await authenticate();
      // Visit the list once while online so the IndexedDB cache is primed and
      // the SyncStatusBanner's useOnlineSync hook is mounted.
      await page.goto('/persons');
      await page.getByRole('button', { name: 'Nova Pessoa' }).click();
      // The name <input> has no htmlFor-associated label, so target it by its
      // placeholder (the only stable, unique handle on the create form).
      await page.getByPlaceholder('Ex: Maria da Silva Santos').fill(personName);

      // Go offline BEFORE submit: the create must persist to Dexie + sync queue.
      await context.setOffline(true);
      await page.getByRole('button', { name: 'Cadastrar Pessoa' }).click();

      // The offline create navigates back to the list via client-side routing
      // (no full reload — a hard reload offline would fail to re-auth). The
      // person is served from the IndexedDB cache + pending offline record.
      await expect(page).toHaveURL(/\/persons$/);
      await expect(page.getByText(personName)).toBeVisible();

      // Reconnect: the drainer flushes the queue to /api/v1/sync/push.
      await context.setOffline(false);

      // Poll for the synced state: the row reaches Postgres.
      await expect
        .poll(
          async () =>
            withClient(async (client) => {
              const res = await client.query(`SELECT 1 FROM person WHERE full_name = $1`, [
                personName,
              ]);
              return res.rowCount ?? 0;
            }),
          { timeout: 30_000, message: 'sync queue should drain to Postgres after reconnect' },
        )
        .toBe(1);
    },
  );

  // ── OFFLINE TRIAGE SLICE — exercises the S05.2 per-entity offline-create path.
  // A person is created ONLINE first (to obtain a person_id), then a triage is
  // created OFFLINE: it must persist to Dexie + the sync queue, render from the
  // cached triage list, and drain to Postgres on reconnect.
  test(
    '@smoke offline triage stays visible then drains on reconnect',
    async ({ page, authenticate, context, identity }) => {
      const personName = `${identity.dataPrefix}Triage Person`;
      const complaint = `${identity.dataPrefix}dor nas costas`;

      await authenticate();

      // Create the person online so the triage form has a person_id.
      await page.goto('/persons');
      await page.getByRole('button', { name: 'Nova Pessoa' }).click();
      await page.getByPlaceholder('Ex: Maria da Silva Santos').fill(personName);
      await page.getByRole('button', { name: 'Cadastrar Pessoa' }).click();
      await expect(page).toHaveURL(/\/persons\/[0-9a-f-]{36}/);
      const personId = page.url().split('/persons/')[1] ?? '';

      // Prime the triage list cache + mount the drainer while online.
      await page.goto('/triages');

      // Open the triage form WHILE ONLINE so its person + service-type context
      // loads (the create page fetches those on mount). Then fill and go offline
      // before submit — mirroring the person slice: the create itself is offline.
      await page.goto(`/triages/new?person_id=${personId}`);
      const complaintField = page.locator('textarea').first();
      await expect(complaintField).toBeVisible();
      await complaintField.fill(complaint);

      await context.setOffline(true);
      await page.getByRole('button', { name: 'Salvar Triagem' }).click();

      // Offline create returns to the list, served from the IndexedDB cache.
      await expect(page).toHaveURL(/\/triages$/);
      await expect(page.getByText(complaint)).toBeVisible();

      // Reconnect: the drainer flushes the queued triage to /api/v1/sync/push.
      await context.setOffline(false);

      await expect
        .poll(
          async () =>
            withClient(async (client) => {
              const res = await client.query(
                `SELECT 1 FROM triage WHERE main_complaint = $1`,
                [complaint],
              );
              return res.rowCount ?? 0;
            }),
          { timeout: 30_000, message: 'triage sync queue should drain to Postgres after reconnect' },
        )
        .toBe(1);
    },
  );
});
