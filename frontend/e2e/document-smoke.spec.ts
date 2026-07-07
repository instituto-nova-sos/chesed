import { test, expect, DATABASE_URL, withClient, createPerson } from './fixtures';

/**
 * @smoke — document upload critical slice (E08, S08.2 / RF-06, RF-30).
 *
 * Proves end-to-end (screen → real Go API → MinIO object storage → Postgres):
 *   1. create a person online
 *   2. upload a PDF via the person's document section
 *   3. the document metadata row lands in Postgres, campus-scoped
 *
 * The uploaded object itself goes to the E2E MinIO bucket; here we assert the
 * metadata row + type, which proves the multipart POST reached storage+DB.
 */

// Minimal valid PDF (header + EOF) so the backend magic-byte check passes.
const PDF_BYTES = Buffer.from(
  '%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n',
  'utf8',
);

test.describe('document critical slice', () => {
  test('@smoke upload document → DB', async ({ page, authenticate, identity }) => {
    const personName = `${identity.dataPrefix}Diego Documento`;
    const fileName = `${identity.dataPrefix}rg.pdf`;

    await authenticate();
    const personId = await createPerson(page, personName);

    await page.goto(`/persons/${personId}`);
    await expect(page.getByRole('heading', { name: 'Documentos' })).toBeVisible();
    await page.getByRole('button', { name: 'Enviar documento' }).click();

    // Modal fields.
    await expect(page.getByRole('heading', { name: 'Enviar Documento' })).toBeVisible();
    await page.getByLabel('Arquivo do documento').setInputFiles({
      name: fileName,
      mimeType: 'application/pdf',
      buffer: PDF_BYTES,
    });
    await page.getByLabel('Tipo de documento').selectOption('ID');

    // Exact match so this targets the modal submit "Enviar" and not the section
    // opener "Enviar documento" (both are buttons).
    await page.getByRole('button', { name: 'Enviar', exact: true }).click();

    // Modal closes on success.
    await expect(page.getByRole('heading', { name: 'Enviar Documento' })).toBeHidden();

    // DB: the metadata row exists for the person, campus-scoped, type ID.
    await withClient(async (client) => {
      const res = await client.query<{
        document_type: string;
        file_name: string;
        campus_id: string;
      }>(
        `SELECT document_type, file_name, campus_id FROM document
         WHERE person_id = $1 AND file_name = $2`,
        [personId, fileName],
      );
      expect(res.rowCount, `document row should exist in ${DATABASE_URL}`).toBe(1);
      expect(res.rows[0]?.document_type).toBe('ID');
      expect(res.rows[0]?.campus_id).toBe(identity.campusId);
    });
  });
});
