/**
 * Integration test: consents API surface against MSW.
 *
 * Exercises the real `apiClient` path for the Sprint 6 consent contract
 * (docs/11-api-design.md): POST body shape (optional guardian omitted when
 * unset), the list envelope, the PATCH revoke path and body, ApiError
 * mapping for the 409 duplicate, and the `usePersonConsents` hook flow.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { createConsent, listPersonConsents, revokeConsent } from '../api/consents';
import { ApiError } from '../api/client';
import { usePersonConsents } from '../hooks/usePersonConsents';
import type { Consent } from '../types/consent';
import { API_BASE, server } from './server';
import './setup';

const PERSON_ID = '00000000-0000-0000-0000-0000000000a1';
const CONSENT_ID = '00000000-0000-0000-0000-0000000000c1';

function consentFixture(overrides: Partial<Consent> = {}): Consent {
  return {
    id: CONSENT_ID,
    person_id: PERSON_ID,
    consent_type: 'DATA_PROCESSING',
    consent_version: '1.0',
    purpose: 'Cadastro e acompanhamento de atendimentos',
    granted_at: '2026-07-03T14:30:00Z',
    granted_by_person_id: null,
    signature_data: 'data:image/png;base64,STUB',
    is_active: true,
    revoked_at: null,
    revoked_reason: null,
    ...overrides,
  };
}

describe('integration: consent create + MSW', () => {
  it('posts the documented create body without an unset guardian', async () => {
    let seenBody: Record<string, unknown> | null = null;
    server.use(
      http.post(`${API_BASE}/consents`, async ({ request }) => {
        seenBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json(consentFixture(), { status: 201 });
      }),
    );

    const created = await createConsent({
      person_id: PERSON_ID,
      consent_type: 'DATA_PROCESSING',
      purpose: 'Cadastro e acompanhamento de atendimentos',
      consent_version: '1.0',
      signature_data: 'data:image/png;base64,STUB',
    });

    expect(created.id).toBe(CONSENT_ID);
    expect(created.is_active).toBe(true);
    expect(seenBody).toMatchObject({
      person_id: PERSON_ID,
      consent_type: 'DATA_PROCESSING',
      purpose: 'Cadastro e acompanhamento de atendimentos',
      signature_data: 'data:image/png;base64,STUB',
    });
    expect(Object.keys(seenBody ?? {})).not.toContain('granted_by_person_id');
  });

  it('includes granted_by_person_id when a guardian is set', async () => {
    let seenBody: Record<string, unknown> | null = null;
    server.use(
      http.post(`${API_BASE}/consents`, async ({ request }) => {
        seenBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json(
          consentFixture({ consent_type: 'MINOR_GUARDIAN' }),
          { status: 201 },
        );
      }),
    );

    await createConsent({
      person_id: PERSON_ID,
      consent_type: 'MINOR_GUARDIAN',
      purpose: 'Autorização do responsável legal',
      granted_by_person_id: '00000000-0000-0000-0000-0000000000b2',
    });

    expect(seenBody).toMatchObject({
      granted_by_person_id: '00000000-0000-0000-0000-0000000000b2',
    });
  });

  it('maps a 409 duplicate create to an ApiError with status 409', async () => {
    server.use(
      http.post(`${API_BASE}/consents`, () =>
        HttpResponse.json({ error: 'duplicate' }, { status: 409 }),
      ),
    );

    const attempt = createConsent({
      person_id: PERSON_ID,
      consent_type: 'DATA_PROCESSING',
      purpose: 'Cadastro',
    });

    await expect(attempt).rejects.toBeInstanceOf(ApiError);
    await expect(
      createConsent({ person_id: PERSON_ID, consent_type: 'DATA_PROCESSING', purpose: 'Cadastro' }),
    ).rejects.toMatchObject({ name: 'ApiError', status: 409 });
  });
});

describe('integration: consent list and revoke + MSW', () => {
  it('maps the list envelope into Consent objects', async () => {
    server.use(
      http.get(`${API_BASE}/persons/:id/consents`, ({ params }) =>
        HttpResponse.json({
          data: [
            consentFixture(),
            consentFixture({
              id: '00000000-0000-0000-0000-0000000000c2',
              person_id: params.id as string,
              consent_type: 'IMAGE_USAGE',
              is_active: false,
              revoked_at: '2026-07-01T10:00:00Z',
              revoked_reason: 'Pedido do titular',
            }),
          ],
        }),
      ),
    );

    const result = await listPersonConsents(PERSON_ID);

    expect(result.data).toHaveLength(2);
    expect(result.data[0]?.consent_type).toBe('DATA_PROCESSING');
    expect(result.data[1]?.is_active).toBe(false);
    expect(result.data[1]?.revoked_reason).toBe('Pedido do titular');
  });

  it('patches the revoke path with the reason body', async () => {
    let seenId: string | null = null;
    let seenBody: Record<string, unknown> | null = null;
    server.use(
      http.patch(`${API_BASE}/consents/:id/revoke`, async ({ params, request }) => {
        seenId = params.id as string;
        seenBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json(
          consentFixture({
            is_active: false,
            revoked_at: '2026-07-03T15:00:00Z',
            revoked_reason: 'Pedido do titular',
          }),
        );
      }),
    );

    const revoked = await revokeConsent(CONSENT_ID, 'Pedido do titular');

    expect(seenId).toBe(CONSENT_ID);
    expect(seenBody).toEqual({ revoked_reason: 'Pedido do titular' });
    expect(revoked.is_active).toBe(false);
  });
});

describe('integration: usePersonConsents + MSW', () => {
  it('loads the consent registry and refreshes it after a revoke', async () => {
    let listCalls = 0;
    server.use(
      http.get(`${API_BASE}/persons/:id/consents`, () => {
        listCalls += 1;
        return HttpResponse.json({ data: [consentFixture()] });
      }),
      http.patch(`${API_BASE}/consents/:id/revoke`, () =>
        HttpResponse.json(consentFixture({ is_active: false })),
      ),
    );

    const { result } = renderHook(() => usePersonConsents(PERSON_ID));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.consents).toHaveLength(1);
    expect(listCalls).toBe(1);

    await act(() => result.current.revoke(CONSENT_ID, 'Pedido do titular'));

    await waitFor(() => expect(listCalls).toBe(2));
  });
});
