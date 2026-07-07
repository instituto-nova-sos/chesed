import { test, expect, DATABASE_URL, withClient, createPerson } from './fixtures';

/**
 * @smoke — LGPD consent capture critical slice (E08, S08.3 / RF-07, RF-57, RF-58b/c).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. create a person online
 *   2. open the consent form for that person (personId from the URL)
 *   3. capture a DATA_PROCESSING consent with a drawn signature
 *   4. the active consent row lands in Postgres, campus-scoped, with signature data
 *   5. the consent shows in the person's consent history
 */

test.describe('consent critical slice', () => {
  test('@smoke capture consent with signature → DB', async ({
    page,
    authenticate,
    identity,
  }) => {
    const personName = `${identity.dataPrefix}Carla Consentimento`;
    const purpose = `${identity.dataPrefix}tratamento de dados cadastrais`;

    await authenticate();
    const personId = await createPerson(page, personName);

    await page.goto(`/persons/${personId}/consents/new`);
    await expect(page.getByRole('heading', { name: 'Novo consentimento' })).toBeVisible();

    // Type defaults to DATA_PROCESSING; overwrite the prefilled purpose with a
    // test-prefixed value so the row is uniquely identifiable.
    await page.locator('#consent-purpose').fill(purpose);

    // Draw a non-empty signature on the capture canvas (empty → validation error).
    const canvas = page.getByRole('img', { name: 'Assinatura' });
    const box = await canvas.boundingBox();
    if (!box) throw new Error('signature canvas not found');
    await page.mouse.move(box.x + 20, box.y + 20);
    await page.mouse.down();
    await page.mouse.move(box.x + box.width - 20, box.y + box.height - 20, { steps: 8 });
    await page.mouse.up();

    await page.getByRole('button', { name: 'Salvar' }).click();

    // Success navigates back to the person detail page.
    await expect(page).toHaveURL(new RegExp(`/persons/${personId}$`));

    // The consent shows in the person's consent history.
    await expect(page.getByRole('heading', { name: 'Consentimentos' })).toBeVisible();

    // DB: an active DATA_PROCESSING consent exists for the person, campus-scoped,
    // with a non-empty signature payload.
    await withClient(async (client) => {
      const res = await client.query<{
        consent_type: string;
        is_active: boolean;
        campus_id: string;
        signature_data: string | null;
      }>(
        `SELECT consent_type, is_active, campus_id, signature_data
         FROM consent WHERE person_id = $1 AND purpose = $2`,
        [personId, purpose],
      );
      expect(res.rowCount, `consent row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.consent_type).toBe('DATA_PROCESSING');
      expect(res.rows[0]?.is_active).toBe(true);
      expect(res.rows[0]?.campus_id).toBe(identity.campusId);
      expect((res.rows[0]?.signature_data ?? '').length).toBeGreaterThan(0);
    });
  });
});
