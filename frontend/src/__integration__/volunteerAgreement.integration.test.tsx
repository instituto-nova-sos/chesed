/**
 * Integration test: volunteer-agreement API client against an MSW-backed surface.
 *
 * What this exercises that a unit test does not:
 * - The real `apiClient` / `apiClientRaw` HTTP path (fetch + auth header + JSON /
 *   multipart body construction + ApiError mapping).
 * - The request contract for the two self-service accept paths: the JSON
 *   `signature_data` body on `acceptAgreement`, and the multipart `document`
 *   field + endpoint on `uploadAgreementSelf`.
 */

import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { acceptAgreement, uploadAgreementSelf } from '../api/persons';
import { API_BASE, server } from './server';
import './setup';

describe('integration: volunteer agreement API client + MSW', () => {
  it('sends signature_data in the accept body', async () => {
    let received: unknown;
    server.use(
      http.post(`${API_BASE}/volunteer-agreement/accept`, async ({ request }) => {
        received = await request.json();
        return HttpResponse.json({ id: 'a1', status: 'ACCEPTED', signature_method: 'DIGITAL' });
      }),
    );

    const result = await acceptAgreement('data:image/png;base64,SIG');

    expect(received).toEqual({ signature_data: 'data:image/png;base64,SIG' });
    expect(result.status).toBe('ACCEPTED');
  });

  it('uploads the document as multipart to the self-service endpoint', async () => {
    let hasDocument = false;
    let contentType: string | undefined;
    server.use(
      http.post(`${API_BASE}/volunteer-agreement/upload`, async ({ request }) => {
        contentType = request.headers.get('content-type') ?? undefined;
        const form = await request.formData();
        // undici in Node does not preserve File.name (it becomes "blob"), so assert
        // on the contract that matters: the `document` field is present as a blob.
        hasDocument = form.get('document') != null;
        return HttpResponse.json({ id: 'a1', status: 'ACCEPTED', signature_method: 'MANUAL_UPLOAD' });
      }),
    );

    const file = new File(['%PDF-1.4'], 'termo.pdf', { type: 'application/pdf' });
    const result = await uploadAgreementSelf(file);

    expect(contentType).toMatch(/multipart\/form-data/);
    expect(hasDocument).toBe(true);
    expect(result.status).toBe('ACCEPTED');
  });

  it('maps a server error to ApiError', async () => {
    server.use(
      http.post(`${API_BASE}/volunteer-agreement/accept`, () =>
        HttpResponse.json({ error: 'validation', message: 'signature is required' }, { status: 400 }),
      ),
    );

    await expect(acceptAgreement('')).rejects.toMatchObject({ status: 400 });
  });
});
