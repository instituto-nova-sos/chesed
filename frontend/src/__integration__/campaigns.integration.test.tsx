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
  addTeamMember,
  createCampaign,
  getCampaign,
  getCampaignMetrics,
  removeTeamMember,
  updateCampaign,
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

describe('integration: campaign detail/update/team + MSW (review B2)', () => {
  it('fetches the campaign detail with its team array', async () => {
    server.use(
      http.get(`${API_BASE}/campaigns/:id`, ({ params }) =>
        HttpResponse.json({
          id: params.id,
          name: 'Detail Campaign',
          campaign_type: 'HEALTH',
          status: 'ACTIVE',
          start_date: '2026-07-10T00:00:00Z',
          campus_id: '00000000-0000-0000-0000-000000000010',
          created_at: '2026-07-01T10:00:00Z',
          updated_at: '2026-07-01T10:00:00Z',
          team: [
            {
              person_id: '00000000-0000-0000-0000-000000000002',
              person_name: 'Maria Silva',
              role_in_campaign: 'VOLUNTEER',
              assigned_at: '2026-07-01T11:00:00Z',
            },
          ],
        }),
      ),
    );

    const detail = await getCampaign(CAMPAIGN_ID);
    expect(detail.name).toBe('Detail Campaign');
    expect(detail.team).toHaveLength(1);
    expect(detail.team[0]?.person_name).toBe('Maria Silva');
  });

  it('sends the documented PUT body including status on update', async () => {
    let seenBody: unknown = null;
    server.use(
      http.put(`${API_BASE}/campaigns/:id`, async ({ request, params }) => {
        seenBody = await request.json();
        return HttpResponse.json({
          id: params.id,
          name: 'Updated Campaign',
          campaign_type: 'HEALTH',
          status: 'ACTIVE',
          start_date: '2026-07-10T00:00:00Z',
          campus_id: '00000000-0000-0000-0000-000000000010',
          created_at: '2026-07-01T10:00:00Z',
          updated_at: '2026-07-02T10:00:00Z',
        });
      }),
    );

    const updated = await updateCampaign(CAMPAIGN_ID, {
      name: 'Updated Campaign',
      campaign_type: 'HEALTH',
      start_date: '2026-07-10',
      status: 'ACTIVE',
      coordinator_id: '00000000-0000-0000-0000-000000000009',
    });

    expect(updated.status).toBe('ACTIVE');
    expect(seenBody).toMatchObject({
      name: 'Updated Campaign',
      status: 'ACTIVE',
      coordinator_id: '00000000-0000-0000-0000-000000000009',
    });
  });

});

describe('integration: campaign team membership + MSW (review B2)', () => {
  it('adds a team member and maps the 409 duplicate to an ApiError', async () => {
    let calls = 0;
    server.use(
      http.post(`${API_BASE}/campaigns/:id/team`, async ({ request }) => {
        calls += 1;
        const body = (await request.json()) as { person_id: string; role_in_campaign: string };
        if (calls > 1) {
          return HttpResponse.json({ error: 'duplicate' }, { status: 409 });
        }
        return HttpResponse.json(
          {
            person_id: body.person_id,
            person_name: 'Maria Silva',
            role_in_campaign: body.role_in_campaign,
            assigned_at: '2026-07-01T11:00:00Z',
          },
          { status: 201 },
        );
      }),
    );

    const input = {
      person_id: '00000000-0000-0000-0000-000000000002',
      role_in_campaign: 'VOLUNTEER',
    };
    const member = await addTeamMember(CAMPAIGN_ID, input);
    expect(member.person_name).toBe('Maria Silva');

    await expect(addTeamMember(CAMPAIGN_ID, input)).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
    });
  });
});
