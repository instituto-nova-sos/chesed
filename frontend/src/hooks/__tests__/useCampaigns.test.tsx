import { describe, expect, it, vi, beforeEach } from 'vitest';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { CampaignListResponse } from '../../types';

const listCampaigns = vi.fn();
vi.mock('../../api/campaigns', () => ({
  listCampaigns: (...args: unknown[]) => listCampaigns(...args) as Promise<CampaignListResponse>,
}));

import { useCampaigns } from '../useCampaigns';

function response(names: string[]): CampaignListResponse {
  return {
    data: names.map((name, i) => ({
      id: `00000000-0000-0000-0000-00000000000${i}`,
      name,
      campaign_type: 'SOCIAL_ACTION',
      status: 'ACTIVE',
      start_date: '2026-07-10T00:00:00Z',
    })),
    pagination: { page: 1, per_page: 20, total: names.length, total_pages: 1 },
  };
}

describe('useCampaigns', () => {
  beforeEach(() => {
    listCampaigns.mockReset();
  });

  it('loads campaigns on mount', async () => {
    listCampaigns.mockResolvedValue(response(['March Action']));

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.campaigns).toHaveLength(1);
    expect(result.current.campaigns[0]?.name).toBe('March Action');
    expect(result.current.error).toBeNull();
  });

  it('exposes an error message when the API fails', async () => {
    listCampaigns.mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).not.toBeNull();
    expect(result.current.campaigns).toEqual([]);
  });

  it('filterByStatus refetches with the status param and resets the page', async () => {
    listCampaigns.mockResolvedValue(response([]));

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.filterByStatus('COMPLETED'));
    await waitFor(() =>
      expect(listCampaigns).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: 'COMPLETED', page: 1 }),
      ),
    );
  });

  it('goToPage refetches the requested page', async () => {
    listCampaigns.mockResolvedValue(response([]));

    const { result } = renderHook(() => useCampaigns());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.goToPage(3));
    await waitFor(() =>
      expect(listCampaigns).toHaveBeenLastCalledWith(expect.objectContaining({ page: 3 })),
    );
  });
});
