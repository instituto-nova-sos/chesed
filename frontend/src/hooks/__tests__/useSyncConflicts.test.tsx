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

async function seedSnapshot(entityId: string, fullName: string): Promise<void> {
  await db.conflicts.put({
    entityId,
    entityType: 'person',
    serverData: { id: entityId, full_name: fullName },
    serverUpdatedAt: '2026-05-01T00:00:00Z',
    capturedAt: new Date().toISOString(),
  });
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

  it('snapshotFor returns the captured server version', async () => {
    await seedConflict('a');
    await seedSnapshot('a', 'Server Name');
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    const snap = await result.current.snapshotFor('a');
    expect(snap?.serverData.full_name).toBe('Server Name');
  });

  it('keepServer applies the server data and clears the conflict', async () => {
    const key = await seedConflict('a');
    await seedSnapshot('a', 'Server Name');
    await db.persons.put({
      id: 'a',
      data: { id: 'a', full_name: 'Local Name' },
      syncStatus: 'conflict',
      localCreatedAt: new Date().toISOString(),
    });
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    await act(async () => {
      await result.current.keepServer(key);
    });

    await waitFor(() => expect(result.current.conflicts).toHaveLength(0));
    expect((await db.persons.get('a'))?.data.full_name).toBe('Server Name');
    expect((await db.persons.get('a'))?.syncStatus).toBe('synced');
    expect(await db.conflicts.get('a')).toBeUndefined();
  });

  it('keepLocal re-queues the local record and clears the conflict', async () => {
    const key = await seedConflict('a');
    await seedSnapshot('a', 'Server Name');
    await db.persons.put({
      id: 'a',
      data: { id: 'a', full_name: 'Local Name' },
      syncStatus: 'conflict',
      localCreatedAt: new Date().toISOString(),
    });
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    await act(async () => {
      await result.current.keepLocal(key);
    });

    await waitFor(() => expect(result.current.conflicts).toHaveLength(0));
    expect((await db.syncQueue.get(key))?.conflicted).toBeFalsy();
    expect((await db.persons.get('a'))?.syncStatus).toBe('pending');
    expect(await db.conflicts.get('a')).toBeUndefined();
  });

  it('merge stages the merged data and clears the conflict', async () => {
    const key = await seedConflict('a');
    await seedSnapshot('a', 'Server Name');
    await db.persons.put({
      id: 'a',
      data: { id: 'a', full_name: 'Local Name' },
      syncStatus: 'conflict',
      localCreatedAt: new Date().toISOString(),
    });
    const { result } = renderHook(() => useSyncConflicts());
    await waitFor(() => expect(result.current.conflicts).toHaveLength(1));

    await act(async () => {
      await result.current.merge(key, { id: 'a', full_name: 'Merged Name' });
    });

    await waitFor(() => expect(result.current.conflicts).toHaveLength(0));
    expect((await db.syncQueue.get(key))?.data.full_name).toBe('Merged Name');
    expect((await db.persons.get('a'))?.syncStatus).toBe('pending');
    expect(await db.conflicts.get('a')).toBeUndefined();
  });
});
