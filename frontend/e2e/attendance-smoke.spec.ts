import { test, expect, DATABASE_URL, withClient, createPerson } from './fixtures';

/**
 * @smoke — attendance + workflow-transition critical slice
 * (E04, S04.2/S04.3 / RF-27, RF-32..RF-34, RF-35a).
 *
 * Proves end-to-end (screen → real Go API → real Postgres):
 *   1. create a person online
 *   2. schedule an attendance for that person (service type required)
 *   3. walk the workflow SCHEDULED → IN_PROGRESS → COMPLETED via the detail-page
 *      action buttons (POST /attendances/:id/transitions)
 *   4. the attendance row reflects the final status, and the transition history
 *      records both hops
 */

test.describe('attendance critical slice', () => {
  test('@smoke schedule → transition walk → DB', async ({
    page,
    authenticate,
    identity,
  }) => {
    const personName = `${identity.dataPrefix}Bruno Atendimento`;

    await authenticate();
    const personId = await createPerson(page, personName);

    // Schedule the attendance. Service type is the only required select.
    await page.goto(`/attendances/new?person_id=${personId}`);
    await expect(page.getByRole('heading', { name: 'Novo Atendimento' })).toBeVisible();
    await page
      .locator('#attendance-service-type')
      .selectOption({ index: 1 }); // first real service type (index 0 = placeholder)
    await page.getByRole('button', { name: 'Agendar Atendimento' }).click();
    await expect(page).toHaveURL(/\/attendances\/[0-9a-f-]{36}/);

    const attendanceId = new URL(page.url()).pathname.split('/').pop() as string;

    // SCHEDULED → IN_PROGRESS. The "Concluir" action button only renders once the
    // attendance is IN_PROGRESS, so its appearance is the unambiguous signal the
    // first transition round-tripped (the status text appears in both the badge
    // and the history list, which would be a strict-mode violation).
    await page.getByRole('button', { name: 'Iniciar' }).click();
    await expect(page.getByRole('button', { name: 'Concluir' })).toBeVisible();

    // IN_PROGRESS → COMPLETED. Once completed, no action buttons remain, so the
    // "Concluir" button disappearing signals the second transition round-tripped.
    await page.getByRole('button', { name: 'Concluir' }).click();
    await expect(page.getByRole('button', { name: 'Concluir' })).toBeHidden();

    // DB: final status is COMPLETED, campus-scoped.
    await withClient(async (client) => {
      const res = await client.query<{ status: string; campus_id: string }>(
        `SELECT status, campus_id FROM attendance WHERE id = $1`,
        [attendanceId],
      );
      expect(res.rowCount, `attendance row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.status).toBe('COMPLETED');
      expect(res.rows[0]?.campus_id).toBe(identity.campusId);
    });

    // DB: both transitions recorded, in order (cascades on attendance delete).
    await withClient(async (client) => {
      const res = await client.query<{ from_status: string; to_status: string }>(
        `SELECT from_status, to_status FROM attendance_transition
         WHERE attendance_id = $1 ORDER BY transitioned_at`,
        [attendanceId],
      );
      expect(res.rowCount, 'two transition rows').toBe(2);
      expect(res.rows[0]).toMatchObject({ from_status: 'SCHEDULED', to_status: 'IN_PROGRESS' });
      expect(res.rows[1]).toMatchObject({ from_status: 'IN_PROGRESS', to_status: 'COMPLETED' });
    });
  });
});
