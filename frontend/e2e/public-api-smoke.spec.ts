import { test, expect, TEST_CAMPUS_ID, DATABASE_URL, withClient } from './fixtures';

/**
 * @smoke — public WordPress integration API (E12, S12.1 / RNF-20).
 *
 * This surface is UNAUTHENTICATED and has no UI — it is consumed by the public
 * WordPress site. Driven via direct HTTP (no token), against the real Go API on
 * the E2E stack, so it exercises the PublicRateLimit → PublicCampusValidator →
 * PublicCampusTx (non-owner RLS pool) chain end-to-end.
 *
 * Proves:
 *   1. GET /public/campaigns requires a valid campus_id and returns a paginated
 *      list shape with no auth.
 *   2. POST /public/volunteer-signup creates a person + PENDING volunteer
 *      agreement in Postgres, campus-scoped, and returns a PII-minimal body.
 */

const API = process.env.E2E_API_BASE_URL ?? 'http://localhost:8081/api/v1';

// Unique per-run scope so the created person/agreement are swept. Mirrors the
// fixtures' dataPrefix convention (this spec drives HTTP, not the identity form).
const PREFIX = `E2E_public_${Date.now().toString(36)}_`;

test.describe('public API critical slice', () => {
  test('@smoke GET /public/campaigns requires campus_id and returns a list', async ({
    request,
  }) => {
    // Missing campus_id → 400.
    const missing = await request.get(`${API}/public/campaigns`);
    expect(missing.status(), 'missing campus_id → 400').toBe(400);

    // Valid campus_id → 200 with a paginated shape (data + pagination).
    const ok = await request.get(`${API}/public/campaigns?campus_id=${TEST_CAMPUS_ID}`);
    expect(ok.status(), 'valid campus_id → 200').toBe(200);
    const body = await ok.json();
    expect(Array.isArray(body.data), 'response has a data array').toBe(true);
    expect(body.pagination, 'response has pagination').toBeTruthy();
  });

  test('@smoke POST /public/volunteer-signup creates person + PENDING agreement', async ({
    request,
  }) => {
    const fullName = `${PREFIX}Voluntário Público`;
    const res = await request.post(`${API}/public/volunteer-signup`, {
      data: {
        full_name: fullName,
        email: `${PREFIX}vol@example.org`,
        campus_id: TEST_CAMPUS_ID,
      },
    });
    expect(res.status(), 'signup → 201').toBe(201);
    const body = await res.json();
    // PII-minimal response: id, full_name, campus_id, created_at only.
    expect(body.id).toBeTruthy();
    expect(body.full_name).toBe(fullName);
    expect(body.campus_id).toBe(TEST_CAMPUS_ID);

    try {
      // DB: person created on the test campus, with a PENDING volunteer agreement.
      await withClient(async (client) => {
        const person = await client.query<{ id: string; campus_id: string }>(
          `SELECT id, campus_id FROM person WHERE full_name = $1`,
          [fullName],
        );
        expect(person.rowCount, `person row should exist in ${DATABASE_URL}`).toBe(1);
        expect(person.rows[0]?.campus_id).toBe(TEST_CAMPUS_ID);

        const agreement = await client.query<{ status: string }>(
          `SELECT status FROM volunteer_agreement WHERE person_id = $1`,
          [person.rows[0]?.id],
        );
        expect(agreement.rowCount, 'a volunteer_agreement row exists').toBe(1);
        expect(agreement.rows[0]?.status).toBe('PENDING');
      });
    } finally {
      // This spec creates data outside the identity fixture's prefix, so clean up
      // its own rows here (FK order: agreement → role → person).
      await withClient(async (client) => {
        const persons = `SELECT id FROM person WHERE full_name LIKE $1`;
        await client.query(
          `DELETE FROM volunteer_agreement WHERE person_id IN (${persons})`,
          [`${PREFIX}%`],
        );
        await client.query(
          `DELETE FROM person_role WHERE person_id IN (${persons})`,
          [`${PREFIX}%`],
        );
        await client.query(`DELETE FROM person WHERE full_name LIKE $1`, [`${PREFIX}%`]);
      });
    }
  });
});
