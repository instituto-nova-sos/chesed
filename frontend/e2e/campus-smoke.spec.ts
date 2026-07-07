import { test, expect, DATABASE_URL, withClient } from './fixtures';

/**
 * @smoke — campus management critical slice (E11, S09.x / RNF-07, RNF-15).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. ADMIN opens the campus list and creates a new campus
 *   2. the campus row lands in Postgres with the submitted region/country
 *   3. the new campus appears in the list
 *
 * The test campus is deleted last in cleanup (scoped by name prefix), never the
 * migration-seeded default campus.
 */

test.describe('campus critical slice', () => {
  test('@smoke create campus → list → DB', async ({ page, authenticate, identity }) => {
    const campusName = `${identity.dataPrefix}Campus Zona Norte`;

    await authenticate();

    await page.goto('/campuses');
    await expect(page.getByRole('heading', { name: 'Campus' })).toBeVisible();
    await page.getByRole('link', { name: 'Novo Campus' }).click();
    await expect(page).toHaveURL(/\/campuses\/new/);

    await page.getByLabel('Nome *').fill(campusName);
    await page.getByLabel('Região *').selectOption('BRAZIL');
    await page.getByLabel('Cidade').fill('São Paulo');
    await page.getByLabel('Estado').fill('SP');
    await page.getByLabel('País (código ISO) *').fill('BRA');
    await page.getByRole('button', { name: 'Criar Campus' }).click();

    // Back to the list, the new campus is visible.
    await expect(page).toHaveURL(/\/campuses$/);
    await expect(page.getByText(campusName)).toBeVisible();

    // DB: the row exists with the submitted region + country.
    await withClient(async (client) => {
      const res = await client.query<{ region: string; country: string; is_active: boolean }>(
        `SELECT region, country, is_active FROM campus WHERE name = $1`,
        [campusName],
      );
      expect(res.rowCount, `campus row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.region).toBe('BRAZIL');
      expect(res.rows[0]?.country).toBe('BRA');
      expect(res.rows[0]?.is_active).toBe(true);
    });
  });
});
