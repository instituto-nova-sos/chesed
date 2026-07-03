/**
 * Integration test: campaigns API surface against MSW.
 *
 * Exercises the real `apiClient` path for the Sprint 5 campaign contract:
 * query-string serialization for list filters, POST body shape, the 204
 * team-removal response (no JSON body), ApiError mapping for 404 metrics,
 * and the `useCampaigns` hook state flow.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  createCampaign,
  getCampaignMetrics,
  removeTeamMember,
} from '../api/campaigns';
import { ApiError } from '../api/client';
import { useCampaigns } from '../hooks/useCampaigns';
import { API_BASE, server } from './server';
import './setup';

const CAMPAIGN_ID = '00000000-0000-0000-0000-00000000c001';

describe('integration: campaigns + apiClient + MSW', () => {
  it('loads the default campaign list on mount', async () => {
    const { result } = renderHook(() => useCampaigns());

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.campaigns).toHaveLength(1);
    expect(result.current.campaigns[0]?.name).toBe('Default Campaign');
    expect(result.current.pagination.total).toBe(1);
  });

  it('forwards the status filter as a query parameter', async () => {
    let seenStatus: string | null = null;
    server.use(
      http.get(`${API_BASE}/campaigns`, ({ request }) => {
        seenStatus = new URL(request.url).searchParams.get('status');
        return HttpResponse.json({
          data: [],
          pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
        });
      }),
    );

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.filterByStatus('ACTIVE'));
    await waitFor(() => expect(seenStatus).toBe('ACTIVE'));
  });

  it('surfaces a server error through the hook state', async () => {
    server.use(
      http.get(`${API_BASE}/campaigns`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).not.toBeNull();
    expect(result.current.campaigns).toEqual([]);
  });

});

describe('integration: campaign API functions + MSW', () => {
  it('posts the documented create body and returns the created campaign', async () => {
    let seenBody: unknown = null;
    server.use(
      http.post(`${API_BASE}/campaigns`, async ({ request }) => {
        seenBody = await request.json();
        return HttpResponse.json(
          {
            id: CAMPAIGN_ID,
            name: 'March Social Action',
            campaign_type: 'SOCIAL_ACTION',
            status: 'PLANNED',
            start_date: '2026-07-10T00:00:00Z',
            campus_id: '00000000-0000-0000-0000-000000000010',
            created_at: '2026-07-01T10:00:00Z',
            updated_at: '2026-07-01T10:00:00Z',
          },
          { status: 201 },
        );
      }),
    );

    const created = await createCampaign({
      name: 'March Social Action',
      campaign_type: 'SOCIAL_ACTION',
      start_date: '2026-07-10',
    });

    expect(created.status).toBe('PLANNED');
    expect(seenBody).toMatchObject({
      name: 'March Social Action',
      campaign_type: 'SOCIAL_ACTION',
      start_date: '2026-07-10',
    });
  });

  it('handles the 204 team-removal response without a JSON body', async () => {
    server.use(
      http.delete(
        `${API_BASE}/campaigns/${CAMPAIGN_ID}/team/:personId`,
        () => new HttpResponse(null, { status: 204 }),
      ),
    );

    await expect(
      removeTeamMember(CAMPAIGN_ID, '00000000-0000-0000-0000-000000000002'),
    ).resolves.toBeUndefined();
  });

  it('maps a 404 metrics response to an ApiError with status', async () => {
    server.use(
      http.get(`${API_BASE}/reports/campaigns/:id`, () =>
        HttpResponse.json({ error: 'not_found' }, { status: 404 }),
      ),
    );

    await expect(getCampaignMetrics(CAMPAIGN_ID)).rejects.toMatchObject({
      name: 'ApiError',
      status: 404,
    });
    await expect(getCampaignMetrics(CAMPAIGN_ID)).rejects.toBeInstanceOf(ApiError);
  });
});
