import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest';
import 'fake-indexeddb/auto';
import { renderHook, act, waitFor } from '@testing-library/react';
import { db } from '../../offline/db';

// Mock the sync API client so the hook can be tested without a network.
const syncPush = vi.fn();
const syncPull = vi.fn();
vi.mock('../../api/sync', () => ({
  syncPush: (...args: unknown[]) => syncPush(...args),
  syncPull: (...args: unknown[]) => syncPull(...args),
}));

import { useOnlineSync } from '../useOnlineSync';

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

async function enqueuePerson(id: string, name: string) {
  await db.persons.put({
    id,
    data: { id, full_name: name },
    syncStatus: 'pending',
    localCreatedAt: new Date().toISOString(),
  });
  await db.syncQueue.add({
    entityType: 'person',
    entityId: id,
    action: 'create',
    data: { sync_id: id, full_name: name },
    createdAt: new Date().toISOString(),
    retryCount: 0,
  });
}

describe('useOnlineSync', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    syncPush.mockReset();
    syncPull.mockReset();
    syncPull.mockResolvedValue({ records: [], server_timestamp: '2026-01-01T00:00:00Z', has_more: false });
    setOnline(true);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('reports the pending queue count', async () => {
    await enqueuePerson('p1', 'Maria');
    const { result } = renderHook(() => useOnlineSync({ autoStart: false }));
    await waitFor(() => expect(result.current.pendingCount).toBe(1));
  });

  it('drains the queue when syncNow is called while online', async () => {
    await enqueuePerson('p1', 'Maria');
    syncPush.mockResolvedValue({
      results: [{ sync_id: 'p1', status: 'created', server_id: 'srv-p1' }],
      server_timestamp: '2026-01-01T00:00:00Z',
    });

    const { result } = renderHook(() => useOnlineSync({ autoStart: false }));
    await act(async () => {
      await result.current.syncNow();
    });

    expect(syncPush).toHaveBeenCalledOnce();
    await waitFor(() => expect(result.current.pendingCount).toBe(0));
  });

  it('does not push while offline', async () => {
    await enqueuePerson('p1', 'Maria');
    setOnline(false);

    const { result } = renderHook(() => useOnlineSync({ autoStart: false }));
    await act(async () => {
      await result.current.syncNow();
    });

    expect(syncPush).not.toHaveBeenCalled();
    expect(result.current.isOnline).toBe(false);
  });

  it('surfaces the conflict count after a conflicting push', async () => {
    await enqueuePerson('p1', 'Maria');
    syncPush.mockResolvedValue({
      results: [{ sync_id: 'p1', status: 'conflict', message: 'duplicate' }],
      server_timestamp: '2026-01-01T00:00:00Z',
    });

    const { result } = renderHook(() => useOnlineSync({ autoStart: false }));
    await act(async () => {
      await result.current.syncNow();
    });

    await waitFor(() => expect(result.current.conflictCount).toBe(1));
  });

  it('schedules an automatic retry after the backoff delay when a push fails', async () => {
    await enqueuePerson('p1', 'Maria');
    // First push fails (network), second succeeds.
    syncPush
      .mockRejectedValueOnce(new Error('network down'))
      .mockResolvedValueOnce({
        results: [{ sync_id: 'p1', status: 'created', server_id: 'srv-p1' }],
        server_timestamp: '2026-01-01T00:00:00Z',
      });

    // A tiny retry delay keeps the test fast while still proving the scheduling
    // path (real backoffDelay is exercised by its own unit test).
    const { result } = renderHook(() =>
      useOnlineSync({ autoStart: true, retryDelayMs: () => 10 }),
    );
    await act(async () => {
      await result.current.syncNow();
    });
    expect(syncPush).toHaveBeenCalledTimes(1); // first attempt failed, retry scheduled

    // The scheduled retry fires and the second (succeeding) push runs.
    await waitFor(() => expect(syncPush).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(result.current.pendingCount).toBe(0));
  });

  it('drains automatically when the browser comes back online', async () => {
    await enqueuePerson('p1', 'Maria');
    syncPush.mockResolvedValue({
      results: [{ sync_id: 'p1', status: 'created', server_id: 'srv-p1' }],
      server_timestamp: '2026-01-01T00:00:00Z',
    });

    renderHook(() => useOnlineSync({ autoStart: true }));

    act(() => {
      setOnline(true);
      window.dispatchEvent(new Event('online'));
    });

    await waitFor(() => expect(syncPush).toHaveBeenCalled());
  });
});
