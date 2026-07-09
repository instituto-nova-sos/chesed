import { test, expect, DATABASE_URL, withClient } from './fixtures';
import { randomUUID } from 'node:crypto';

/**
 * @smoke — Volunteer agreement digital signature critical slice (RF-07).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. the acting user has a VOLUNTEER role + a PENDING volunteer_agreement (seeded)
 *   2. open /volunteer-agreement, draw a signature, and accept
 *   3. the agreement flips to ACCEPTED via the DIGITAL method in Postgres, with the
 *      drawn signature stored in signature_data
 *   4. accepted_by_user is the local app_user.id (regression guard for the FK bug
 *      where the Keycloak subject was written instead)
 *
 * The acceptance page forces a Keycloak logout on success, so the assertion reads
 * the DB (the source of truth) rather than depending on the post-logout navigation.
 */

test.describe('volunteer agreement critical slice', () => {
  test('@smoke accept agreement with a drawn signature → DB', async ({
    page,
    authenticate,
    identity,
  }) => {
    // The E2E identity is an ADMIN linked to `professionalPersonId`. Give that
    // person an active VOLUNTEER role and a PENDING agreement so the self-service
    // accept flow has something to accept.
    const personId = identity.professionalPersonId;
    const roleId = randomUUID();
    await withClient(async (client) => {
      await client.query(
        `INSERT INTO person_role (id, person_id, role_type, is_active)
         VALUES ($1, $2, 'VOLUNTEER', TRUE)
         ON CONFLICT (person_id, role_type) DO NOTHING`,
        [roleId, personId],
      );
      await client.query(
        `INSERT INTO volunteer_agreement (id, person_id, person_role_id, campus_id, status, agreement_version)
         VALUES ($1, $2, $3, $4, 'PENDING', 'v1')`,
        [randomUUID(), personId, roleId, identity.campusId],
      );
    });

    await authenticate();

    await page.goto('/volunteer-agreement');
    await expect(page.getByRole('heading', { name: 'Termo de Voluntario' })).toBeVisible();

    // Default method is "Assinar digitalmente"; draw a non-empty signature.
    const canvas = page.getByRole('img', { name: 'Assinatura' });
    const box = await canvas.boundingBox();
    if (!box) throw new Error('signature canvas not found');
    await page.mouse.move(box.x + 20, box.y + 20);
    await page.mouse.down();
    await page.mouse.move(box.x + box.width - 20, box.y + box.height - 20, { steps: 8 });
    await page.mouse.up();

    await page.getByRole('button', { name: 'Aceitar Termo' }).click();

    // Success forces a logout; poll the DB for the accepted row instead of the UI.
    await expect
      .poll(
        async () =>
          withClient(async (client) => {
            const res = await client.query<{ status: string }>(
              `SELECT status FROM volunteer_agreement WHERE person_id = $1`,
              [personId],
            );
            return res.rows[0]?.status ?? '';
          }),
        { timeout: 10_000 },
      )
      .toBe('ACCEPTED');

    // Full assertion: DIGITAL method, non-empty signature, and accepted_by_user =
    // the local app_user.id (NOT the Keycloak subject — FK-bug regression guard).
    await withClient(async (client) => {
      const res = await client.query<{
        signature_method: string;
        signature_data: string | null;
        accepted_by_user: string | null;
        campus_id: string;
      }>(
        `SELECT va.signature_method, va.signature_data, va.accepted_by_user, va.campus_id
         FROM volunteer_agreement va WHERE va.person_id = $1`,
        [personId],
      );
      expect(res.rowCount, `agreement row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.signature_method).toBe('DIGITAL');
      expect((res.rows[0]?.signature_data ?? '').length).toBeGreaterThan(0);
      expect(res.rows[0]?.campus_id).toBe(identity.campusId);

      const appUser = await client.query<{ id: string }>(
        `SELECT id FROM app_user WHERE keycloak_subject_id = $1`,
        [identity.subject],
      );
      expect(res.rows[0]?.accepted_by_user).toBe(appUser.rows[0]?.id);
    });
  });
});
