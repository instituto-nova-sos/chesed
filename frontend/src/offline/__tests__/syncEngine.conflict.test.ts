import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';
import {
  drainQueue,
  getConflictSnapshot,
  getConflicts,
  mergePullRecords,
  resolveKeepLocal,
  resolveKeepServer,
  resolveMerged,
  type PullRecord,
  type PushFn,
  type SyncPushRecord,
} from '../syncEngine';

describe('conflict snapshots (field-level resolution)', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  async function enqueue(entityId: string, data: Record<string, unknown>): Promise<number> {
    await db.persons.put({
      id: entityId,
      data,
      syncStatus: 'pending',
      localCreatedAt: new Date().toISOString(),
    });
    return (await db.syncQueue.add({
      entityType: 'person',
      entityId,
      action: 'update',
      data,
      createdAt: new Date().toISOString(),
      retryCount: 0,
    })) as number;
  }

  it('captures a snapshot when the push conflict carries server_data', async () => {
    await enqueue('s1', { id: 's1', full_name: 'Local' });
    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({
        sync_id: r.sync_id,
        status: 'conflict' as const,
        message: 'stale',
        server_data: { id: 's1', full_name: 'Server' },
        server_updated_at: '2026-02-01T00:00:00Z',
        conflicting_fields: ['full_name'],
      })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    await drainQueue(push);

    expect((await db.persons.get('s1'))?.syncStatus).toBe('conflict');
    const snap = await getConflictSnapshot('s1');
    expect(snap?.serverData.full_name).toBe('Server');
    expect(snap?.serverUpdatedAt).toBe('2026-02-01T00:00:00Z');
    expect(snap?.conflictFields).toEqual(['full_name']);
  });

  it('does not write a snapshot when the conflict carries no server_data', async () => {
    await enqueue('s2', { id: 's2', full_name: 'Local' });
    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({ sync_id: r.sync_id, status: 'conflict' as const })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    await drainQueue(push);
    expect(await getConflictSnapshot('s2')).toBeUndefined();
  });

  it('persists the server value on a pull conflict instead of discarding it', async () => {
    await enqueue('s3', { id: 's3', full_name: 'Local edit' });
    const records: PullRecord[] = [
      {
        entity_type: 'person',
        id: 's3',
        data: { id: 's3', full_name: 'Server edit' },
        updated_at: '2026-03-01T00:00:00Z',
      },
    ];

    const result = await mergePullRecords(records);

    expect(result.conflicts).toBe(1);
    expect((await db.persons.get('s3'))?.syncStatus).toBe('conflict');
    const snap = await getConflictSnapshot('s3');
    expect(snap?.serverData.full_name).toBe('Server edit');
    expect(snap?.serverUpdatedAt).toBe('2026-03-01T00:00:00Z');
  });

  it('flags the queued item on a pull conflict so it is resolvable and not re-drained', async () => {
    const queueId = await enqueue('s3b', { id: 's3b', full_name: 'Local edit' });
    await mergePullRecords([
      {
        entity_type: 'person',
        id: 's3b',
        data: { id: 's3b', full_name: 'Server edit' },
        updated_at: '2026-03-01T00:00:00Z',
      },
    ]);

    // The queued item must be marked conflicted so (a) it surfaces in the
    // resolution UI and (b) the next drain does not re-push it and clobber the
    // server change (last-write-wins data loss).
    const item = await db.syncQueue.get(queueId);
    expect(item?.conflicted).toBe(true);
    expect((await getConflicts()).map((c) => c.entityId)).toContain('s3b');

    const push: PushFn = vi.fn(async () => ({ results: [], server_timestamp: '' }));
    await drainQueue(push);
    expect(push).not.toHaveBeenCalled();
  });

  it('resolveKeepServer applies the server data, clears the queue and snapshot', async () => {
    const queueId = await enqueue('s4', { id: 's4', full_name: 'Local' });
    await db.conflicts.put({
      entityId: 's4',
      entityType: 'person',
      serverData: { id: 's4', full_name: 'Server wins' },
      serverUpdatedAt: '2026-04-01T00:00:00Z',
      capturedAt: new Date().toISOString(),
    });
    await db.syncQueue.update(queueId, { conflicted: true });

    await resolveKeepServer(queueId);

    const cached = await db.persons.get('s4');
    expect(cached?.data.full_name).toBe('Server wins');
    expect(cached?.syncStatus).toBe('synced');
    expect(cached?.serverUpdatedAt).toBe('2026-04-01T00:00:00Z');
    expect(await db.syncQueue.get(queueId)).toBeUndefined();
    expect(await getConflictSnapshot('s4')).toBeUndefined();
  });

  it('resolveMerged stages the merged data as pending and clears the snapshot', async () => {
    const queueId = await enqueue('s5', { id: 's5', full_name: 'Local' });
    await db.conflicts.put({
      entityId: 's5',
      entityType: 'person',
      serverData: { id: 's5', full_name: 'Server' },
      capturedAt: new Date().toISOString(),
    });
    await db.syncQueue.update(queueId, { conflicted: true, retryCount: 3 });

    const merged = { id: 's5', full_name: 'Merged value' };
    await resolveMerged(queueId, merged);

    const item = await db.syncQueue.get(queueId);
    expect(item?.data.full_name).toBe('Merged value');
    expect(item?.conflicted).toBeFalsy();
    expect(item?.retryCount).toBe(0);
    expect((await db.persons.get('s5'))?.data.full_name).toBe('Merged value');
    expect((await db.persons.get('s5'))?.syncStatus).toBe('pending');
    expect(await getConflictSnapshot('s5')).toBeUndefined();
  });

  it('resolveKeepLocal re-queues the local record and clears the snapshot', async () => {
    const queueId = await enqueue('s6', { id: 's6', full_name: 'Local wins' });
    await db.conflicts.put({
      entityId: 's6',
      entityType: 'person',
      serverData: { id: 's6', full_name: 'Server' },
      capturedAt: new Date().toISOString(),
    });
    await db.persons.update('s6', { syncStatus: 'conflict' });
    await db.syncQueue.update(queueId, { conflicted: true, retryCount: 2 });

    await resolveKeepLocal(queueId);

    const item = await db.syncQueue.get(queueId);
    expect(item?.conflicted).toBeFalsy();
    expect(item?.retryCount).toBe(0);
    expect((await db.persons.get('s6'))?.syncStatus).toBe('pending');
    expect(await getConflictSnapshot('s6')).toBeUndefined();
  });
});
