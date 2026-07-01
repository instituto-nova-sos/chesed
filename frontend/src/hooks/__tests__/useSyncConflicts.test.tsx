import { describe, expect, it, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';
import { renderHook, act, waitFor } from '@testing-library/react';
import { db } from '../../offline/db';
import { useSyncConflicts } from '../useSyncConflicts';

async function seedConflict(entityId: string): Promise<number> {
  return (await db.syncQueue.add({
    entityType: 'person',
    entityId,
    action: 'create',
    data: { sync_id: entityId, full_name: entityId },
    createdAt: new Date().toISOString(),
    retryCount: 1,
    conflicted: true,
    lastError: 'conflict',
  })) as number;
}

describe('useSyncConflicts', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('loads the conflicted queue items', async () => {
    await seedConflict('a');
    await seedConflict('b');
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(2));
    expect(result.current.conflicts.map((c) => c.entityId).sort()).toEqual(['a', 'b']);
  });

  it('discard removes the conflict from the list', async () => {
    const key = await seedConflict('a');
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    await act(async () => {
      await result.current.discard(key);
    });

    await waitFor(() => expect(result.current.conflicts).toHaveLength(0));
    expect(await db.syncQueue.count()).toBe(0);
  });

  it('resubmit clears the conflict flag so the item is retryable', async () => {
    const key = await seedConflict('a');
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    await act(async () => {
      await result.current.resubmit(key);
    });

    await waitFor(() => expect(result.current.conflicts).toHaveLength(0));
    const item = await db.syncQueue.get(key);
    expect(item?.conflicted).toBeFalsy();
  });
});
