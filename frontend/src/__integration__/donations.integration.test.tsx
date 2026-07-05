/**
 * Integration test: donations API surface against MSW.
 *
 * Exercises the real `apiClient` path for the E09 donation contract:
 * query-string serialization for list filters, POST body shape for FINANCIAL
 * (with amount) and GOODS (without amount), the PUT update body, detail fetch,
 * ApiError mapping for a 404 detail, and the `useDonations` hook state flow.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  createDonation,
  getDonation,
  listDonations,
  updateDonation,
} from '../api/donations';
import { ApiError } from '../api/client';
import { useDonations } from '../hooks/useDonations';
import { API_BASE, server } from './server';
import './setup';

const DONATION_ID = '00000000-0000-0000-0000-00000000d001';

// The shared server default handler for /donations is owned by the
// orchestrator; register a self-contained default so these tests do not
// depend on its wiring order (setup uses onUnhandledRequest: 'error').
function useDefaultDonationList() {
  server.use(
    http.get(`${API_BASE}/donations`, () =>
      HttpResponse.json({
        data: [
          {
            id: DONATION_ID,
            donation_type: 'FINANCIAL',
            amount: 150,
            currency: 'BRL',
            donation_date: '2026-07-04T00:00:00Z',
            donor_name: 'Maria Silva',
            campaign_name: null,
          },
        ],
        pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
      }),
    ),
  );
}

describe('integration: donations + apiClient + MSW', () => {
  it('loads the default donation list on mount', async () => {
    useDefaultDonationList();
    const { result } = renderHook(() => useDonations());

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.donations).toHaveLength(1);
    expect(result.current.donations[0]?.donor_name).toBe('Maria Silva');
    expect(result.current.pagination.total).toBe(1);
  });

  it('forwards the donation_type filter as a query parameter', async () => {
    let seenType: string | null = null;
    server.use(
      http.get(`${API_BASE}/donations`, ({ request }) => {
        seenType = new URL(request.url).searchParams.get('donation_type');
        return HttpResponse.json({
          data: [],
          pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
        });
      }),
    );

    const { result } = renderHook(() => useDonations());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.filterByType('GOODS'));
    await waitFor(() => expect(seenType).toBe('GOODS'));
  });

  it('surfaces a server error through the hook state', async () => {
    server.use(
      http.get(`${API_BASE}/donations`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    const { result } = renderHook(() => useDonations());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).not.toBeNull();
    expect(result.current.donations).toEqual([]);
  });
});

describe('integration: donation API functions + MSW', () => {
  it('posts a FINANCIAL create body with amount and returns the created donation', async () => {
    let seenBody: unknown = null;
    server.use(
      http.post(`${API_BASE}/donations`, async ({ request }) => {
        seenBody = await request.json();
        return HttpResponse.json(
          {
            id: DONATION_ID,
            donor_person_id: '00000000-0000-0000-0000-000000000002',
            campaign_id: null,
            campus_id: '00000000-0000-0000-0000-000000000010',
            donation_type: 'FINANCIAL',
            amount: 150,
            currency: 'BRL',
            item_description: null,
            donation_date: '2026-07-04T00:00:00Z',
            receipt_number: null,
            receipt_issued_at: null,
            notes: null,
            created_at: '2026-07-04T10:00:00Z',
            updated_at: '2026-07-04T10:00:00Z',
          },
          { status: 201 },
        );
      }),
    );

    const created = await createDonation({
      donation_type: 'FINANCIAL',
      amount: 150,
      currency: 'BRL',
      donor_person_id: '00000000-0000-0000-0000-000000000002',
      donation_date: '2026-07-04',
    });

    expect(created.donation_type).toBe('FINANCIAL');
    expect(created.amount).toBe(150);
    expect(seenBody).toMatchObject({
      donation_type: 'FINANCIAL',
      amount: 150,
      currency: 'BRL',
      donation_date: '2026-07-04',
    });
  });

});

describe('integration: donation create/update bodies + MSW', () => {
  it('posts a GOODS create body without amount', async () => {
    let seenBody: Record<string, unknown> | null = null;
    server.use(
      http.post(`${API_BASE}/donations`, async ({ request }) => {
        seenBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json(
          {
            id: DONATION_ID,
            donor_person_id: null,
            campaign_id: null,
            campus_id: '00000000-0000-0000-0000-000000000010',
            donation_type: 'GOODS',
            amount: null,
            currency: 'BRL',
            item_description: 'Cestas básicas',
            donation_date: '2026-07-04T00:00:00Z',
            receipt_number: null,
            receipt_issued_at: null,
            notes: null,
            created_at: '2026-07-04T10:00:00Z',
            updated_at: '2026-07-04T10:00:00Z',
          },
          { status: 201 },
        );
      }),
    );

    const created = await createDonation({
      donation_type: 'GOODS',
      item_description: 'Cestas básicas',
      donation_date: '2026-07-04',
    });

    expect(created.donation_type).toBe('GOODS');
    expect(created.amount).toBeNull();
    expect(seenBody).toMatchObject({
      donation_type: 'GOODS',
      item_description: 'Cestas básicas',
    });
    expect(seenBody).not.toHaveProperty('amount');
  });

  it('sends the documented PUT body on update', async () => {
    let seenBody: unknown = null;
    server.use(
      http.put(`${API_BASE}/donations/:id`, async ({ request, params }) => {
        seenBody = await request.json();
        return HttpResponse.json({
          id: params.id,
          donor_person_id: null,
          campaign_id: null,
          campus_id: '00000000-0000-0000-0000-000000000010',
          donation_type: 'FINANCIAL',
          amount: 200,
          currency: 'BRL',
          item_description: null,
          donation_date: '2026-07-04T00:00:00Z',
          receipt_number: null,
          receipt_issued_at: null,
          notes: 'Atualizada',
          created_at: '2026-07-04T10:00:00Z',
          updated_at: '2026-07-05T10:00:00Z',
        });
      }),
    );

    const updated = await updateDonation(DONATION_ID, {
      donation_type: 'FINANCIAL',
      amount: 200,
      currency: 'BRL',
      notes: 'Atualizada',
    });

    expect(updated.amount).toBe(200);
    expect(seenBody).toMatchObject({
      donation_type: 'FINANCIAL',
      amount: 200,
      notes: 'Atualizada',
    });
  });

});

describe('integration: donation detail/filter + MSW', () => {
  it('fetches the donation detail with donor and campaign names', async () => {
    server.use(
      http.get(`${API_BASE}/donations/:id`, ({ params }) =>
        HttpResponse.json({
          id: params.id,
          donor_person_id: '00000000-0000-0000-0000-000000000002',
          campaign_id: '00000000-0000-0000-0000-00000000c001',
          campus_id: '00000000-0000-0000-0000-000000000010',
          donation_type: 'FINANCIAL',
          amount: 150,
          currency: 'BRL',
          item_description: null,
          donation_date: '2026-07-04T00:00:00Z',
          receipt_number: null,
          receipt_issued_at: null,
          notes: null,
          created_at: '2026-07-04T10:00:00Z',
          updated_at: '2026-07-04T10:00:00Z',
          donor_name: 'Maria Silva',
          campaign_name: 'Ação Social de Julho',
        }),
      ),
    );

    const detail = await getDonation(DONATION_ID);
    expect(detail.donor_name).toBe('Maria Silva');
    expect(detail.campaign_name).toBe('Ação Social de Julho');
  });

  it('maps a 404 detail response to an ApiError with status', async () => {
    server.use(
      http.get(`${API_BASE}/donations/:id`, () =>
        HttpResponse.json({ error: 'not_found' }, { status: 404 }),
      ),
    );

    await expect(getDonation(DONATION_ID)).rejects.toMatchObject({
      name: 'ApiError',
      status: 404,
    });
    await expect(getDonation(DONATION_ID)).rejects.toBeInstanceOf(ApiError);
  });

  it('serializes the campaign_id list filter as a query parameter', async () => {
    let seenCampaign: string | null = null;
    server.use(
      http.get(`${API_BASE}/donations`, ({ request }) => {
        seenCampaign = new URL(request.url).searchParams.get('campaign_id');
        return HttpResponse.json({
          data: [],
          pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
        });
      }),
    );

    await listDonations({ campaign_id: '00000000-0000-0000-0000-00000000c001' });
    expect(seenCampaign).toBe('00000000-0000-0000-0000-00000000c001');
  });
});
