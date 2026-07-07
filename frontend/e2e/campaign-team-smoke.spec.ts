import { test, expect, DATABASE_URL, withClient, createPerson } from './fixtures';

/**
 * @smoke — campaign team assignment critical slice (E07, S07.2 / RF-38).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. create a person and a campaign online
 *   2. add that person to the campaign team via the detail-page form
 *   3. the campaign_team junction row lands in Postgres with the chosen role
 *
 * Campaign create itself is covered by campaign-smoke; this focuses on the team
 * (person↔campaign) linkage.
 */

test.describe('campaign team critical slice', () => {
  test('@smoke assign person to campaign team → DB', async ({
    page,
    authenticate,
    identity,
  }) => {
    const personName = `${identity.dataPrefix}Elis Equipe`;
    const campaignName = `${identity.dataPrefix}Campanha Inverno`;

    await authenticate();
    const personId = await createPerson(page, personName);

    // Create the campaign.
    await page.goto('/campaigns/new');
    await page.getByLabel('Nome').fill(campaignName);
    await page.getByLabel('Data de início').fill('2026-06-01');
    await page.getByRole('button', { name: 'Salvar' }).click();
    await expect(page).toHaveURL(/\/campaigns\/[0-9a-f-]{36}/);
    const campaignId = new URL(page.url()).pathname.split('/').pop() as string;

    // Add the person to the team via the "Equipe" section's SearchableSelect.
    // The person picker is the combobox under the "Pessoa" label; scope to its
    // group so the selector can't also match the "Função na campanha" <select>
    // (which has an implicit combobox role).
    await expect(page.getByRole('heading', { name: 'Equipe' })).toBeVisible();
    const personPicker = page
      .locator('div.relative', { has: page.getByText('Pessoa', { exact: true }) })
      .getByRole('combobox');
    await personPicker.click();
    await personPicker.fill(personName);
    await page.getByRole('option', { name: personName }).click();
    await page.getByLabel('Função na campanha').selectOption('VOLUNTEER');
    await page.getByRole('button', { name: 'Adicionar' }).click();

    // Wait for the added member to render in the team list. This is the UI signal
    // that the POST /campaign-team round-tripped and persisted, so the DB query
    // below doesn't race ahead of the write.
    await expect(
      page.getByRole('listitem').filter({ hasText: personName }),
    ).toBeVisible();

    // DB: the junction row exists with the chosen role.
    await withClient(async (client) => {
      const res = await client.query<{ role_in_campaign: string }>(
        `SELECT role_in_campaign FROM campaign_team WHERE campaign_id = $1 AND person_id = $2`,
        [campaignId, personId],
      );
      expect(res.rowCount, `campaign_team row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.role_in_campaign).toBe('VOLUNTEER');
    });
  });
});
