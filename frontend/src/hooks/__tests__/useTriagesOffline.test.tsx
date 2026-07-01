import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { renderHook, waitFor } from '@testing-library/react';
import { db } from '../../offline/db';

const listTriages = vi.fn();
vi.mock('../../api/triages', () => ({
  listTriages: (...args: unknown[]) => listTriages(...args),
}));

import { useTriages } from '../useTriages';

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

const serverPage = {
  data: [
    {
      id: 'srv-1',
      person_id: 'p-1',
      person_name: 'Server Person',
      main_complaint: 'Febre',
      triage_date: '2026-06-30',
      requested_service_count: 1,
    },
  ],
  pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
};

describe('useTriages offline fallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    listTriages.mockReset();
    setOnline(true);
  });

  it('caches the server list on a successful online fetch', async () => {
    listTriages.mockResolvedValue(serverPage);
    const { result } = renderHook(() => useTriages());
    await waitFor(() => expect(result.current.triages).toHaveLength(1));
    expect(await db.triages.get('srv-1')).toBeDefined();
  });

  it('shows cached triages (including pending offline records) when offline', async () => {
    await db.triages.bulkPut([
      {
        id: 'srv-1',
        data: { id: 'srv-1', person_name: 'Cached', main_complaint: 'x' },
        syncStatus: 'synced',
        localCreatedAt: 'x',
      },
      {
        id: 'off-1',
        data: { id: 'off-1', person_name: 'My Offline Triage', main_complaint: 'y' },
        syncStatus: 'pending',
        localCreatedAt: 'y',
      },
    ]);
    setOnline(false);
    listTriages.mockRejectedValue(new TypeError('Failed to fetch'));

    const { result } = renderHook(() => useTriages());

    await waitFor(() => expect(result.current.triages.length).toBe(2));
    const names = result.current.triages.map((t) => t.person_name);
    expect(names).toContain('My Offline Triage');
    expect(result.current.error).toBeNull();
  });
});
