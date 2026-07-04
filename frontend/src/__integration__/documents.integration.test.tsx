/**
 * Integration test: documents API surface against MSW.
 *
 * Exercises the real `apiClientRaw`/`apiClient` path for the Sprint 6
 * document contract: multipart FormData field names on upload, the
 * `{ data: [...] }` list envelope through `usePersonDocuments`, the
 * presigned download URL fetch, and ApiError mapping for 400/404.
 */

import { File as NodeFile } from 'node:buffer';
import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { FormData as UndiciFormData } from 'undici';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getDocumentDownloadUrl,
  listPersonDocuments,
  uploadPersonDocument,
} from '../api/documents';
import { ApiError } from '../api/client';
import { usePersonDocuments } from '../hooks/usePersonDocuments';
import type { PersonDocument } from '../types';
import { API_BASE, server } from './server';
import './setup';

const PERSON_ID = '00000000-0000-0000-0000-0000000000d1';
const DOCUMENT_ID = '00000000-0000-0000-0000-0000000000d2';

/**
 * Vitest's jsdom environment exposes jsdom's File/FormData globals while
 * `fetch` is Node's undici; undici does not brand-recognize jsdom's entries
 * and mangles the multipart parts on the wire (filename degrades to "blob",
 * bytes are lost). Browsers are single-realm, so production is unaffected.
 * Upload tests stub FormData with undici's own and build files with
 * node:buffer's File (the same implementation undici uses) so the full
 * multipart contract — field names, filename, bytes — round-trips faithfully.
 */
function uploadFile(name: string, content: string, type: string): File {
  return new NodeFile([content], name, { type }) as unknown as File;
}

function documentPayload(overrides: Partial<PersonDocument> = {}): PersonDocument {
  return {
    id: DOCUMENT_ID,
    person_id: PERSON_ID,
    attendance_id: null,
    document_type: 'PROOF_OF_RESIDENCE',
    file_name: 'comprovante.pdf',
    file_size: 182734,
    mime_type: 'application/pdf',
    description: null,
    uploaded_at: '2026-07-03T14:30:00Z',
    ...overrides,
  };
}

describe('integration: document upload + apiClientRaw + MSW', () => {
  beforeEach(() => {
    vi.stubGlobal('FormData', UndiciFormData);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('posts multipart form-data with the documented field names', async () => {
    let seenFileName: string | null = null;
    let seenFileContent: string | null = null;
    let seenDocumentType: FormDataEntryValue | null = null;
    let seenDescription: FormDataEntryValue | null = null;
    server.use(
      http.post(`${API_BASE}/persons/:id/documents`, async ({ request }) => {
        const form = await request.formData();
        const fileEntry = form.get('document');
        if (fileEntry !== null && typeof fileEntry !== 'string') {
          seenFileName = fileEntry.name;
          seenFileContent = await fileEntry.text();
        }
        seenDocumentType = form.get('document_type');
        seenDescription = form.get('description');
        return HttpResponse.json(
          documentPayload({ description: 'Conta de luz' }),
          { status: 201 },
        );
      }),
    );

    const file = uploadFile('comprovante.pdf', '%PDF-1.4', 'application/pdf');
    const created = await uploadPersonDocument(
      PERSON_ID,
      file,
      'PROOF_OF_RESIDENCE',
      'Conta de luz',
    );

    expect(seenFileName).toBe('comprovante.pdf');
    expect(seenFileContent).toBe('%PDF-1.4');
    expect(seenDocumentType).toBe('PROOF_OF_RESIDENCE');
    expect(seenDescription).toBe('Conta de luz');
    expect(created.id).toBe(DOCUMENT_ID);
    expect(created.document_type).toBe('PROOF_OF_RESIDENCE');
  });

  it('omits the description field when not provided', async () => {
    let hasDescription: boolean | null = null;
    server.use(
      http.post(`${API_BASE}/persons/:id/documents`, async ({ request }) => {
        const form = await request.formData();
        hasDescription = form.has('description');
        return HttpResponse.json(documentPayload(), { status: 201 });
      }),
    );

    const file = uploadFile('comprovante.pdf', '%PDF-1.4', 'application/pdf');
    await uploadPersonDocument(PERSON_ID, file, 'PROOF_OF_RESIDENCE');

    expect(hasDescription).toBe(false);
  });

  it('maps a 400 validation_error to an ApiError with status', async () => {
    server.use(
      http.post(`${API_BASE}/persons/:id/documents`, () =>
        HttpResponse.json({ error: 'validation_error' }, { status: 400 }),
      ),
    );

    const file = uploadFile('nota.txt', 'not a pdf', 'text/plain');
    const attempt = uploadPersonDocument(PERSON_ID, file, 'OTHER');
    await expect(attempt).rejects.toBeInstanceOf(ApiError);
    await expect(
      uploadPersonDocument(PERSON_ID, file, 'OTHER'),
    ).rejects.toMatchObject({ status: 400 });
  });
});

describe('integration: document list + usePersonDocuments + MSW', () => {
  it('maps the { data: [...] } envelope through the hook', async () => {
    server.use(
      http.get(`${API_BASE}/persons/:id/documents`, () =>
        HttpResponse.json({ data: [documentPayload()] }),
      ),
    );

    const { result } = renderHook(() => usePersonDocuments(PERSON_ID));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.documents).toHaveLength(1);
    expect(result.current.documents[0]?.file_name).toBe('comprovante.pdf');
  });

  it('returns the raw envelope from listPersonDocuments', async () => {
    server.use(
      http.get(`${API_BASE}/persons/:id/documents`, () =>
        HttpResponse.json({ data: [documentPayload(), documentPayload({ id: 'x' })] }),
      ),
    );

    const response = await listPersonDocuments(PERSON_ID);
    expect(response.data).toHaveLength(2);
  });

  it('surfaces a server error through the hook state', async () => {
    server.use(
      http.get(`${API_BASE}/persons/:id/documents`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    const { result } = renderHook(() => usePersonDocuments(PERSON_ID));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).not.toBeNull();
    expect(result.current.documents).toEqual([]);
  });

  it('does not fetch when the person id is undefined', async () => {
    let called = false;
    server.use(
      http.get(`${API_BASE}/persons/:id/documents`, () => {
        called = true;
        return HttpResponse.json({ data: [] });
      }),
    );

    const { result } = renderHook(() => usePersonDocuments(undefined));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(called).toBe(false);
  });
});

describe('integration: document download URL + MSW', () => {
  it('fetches the presigned URL with its expiry', async () => {
    server.use(
      http.get(`${API_BASE}/documents/:id/download`, ({ params }) =>
        HttpResponse.json({
          url: `https://storage.example.com/chesed-docs/documents/${String(params.id)}?X-Amz-Signature=abc`,
          expires_at: '2026-07-03T14:45:00Z',
        }),
      ),
    );

    const download = await getDocumentDownloadUrl(DOCUMENT_ID);
    expect(download.url).toContain(DOCUMENT_ID);
    expect(download.expires_at).toBe('2026-07-03T14:45:00Z');
  });

  it('maps a 404 to an ApiError with status', async () => {
    server.use(
      http.get(`${API_BASE}/documents/:id/download`, () =>
        HttpResponse.json({ error: 'not_found' }, { status: 404 }),
      ),
    );

    await expect(getDocumentDownloadUrl(DOCUMENT_ID)).rejects.toBeInstanceOf(ApiError);
    await expect(getDocumentDownloadUrl(DOCUMENT_ID)).rejects.toMatchObject({ status: 404 });
  });
});
