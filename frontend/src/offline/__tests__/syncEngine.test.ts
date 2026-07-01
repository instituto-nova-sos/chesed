import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';
import {
  backoffDelay,
  drainQueue,
  getConflicts,
  getRetryable,
  discardConflict,
  requeueConflict,
  MAX_RETRIES,
  type PushFn,
  type SyncPushRecord,
} from '../syncEngine';

describe('backoffDelay', () => {
  it('follows the 5s → 30s → 2m → 10m schedule, then stays capped', () => {
    expect(backoffDelay(0)).toBe(5_000);
    expect(backoffDelay(1)).toBe(30_000);
    expect(backoffDelay(2)).toBe(120_000);
    expect(backoffDelay(3)).toBe(600_000);
    expect(backoffDelay(4)).toBe(600_000);
    expect(backoffDelay(99)).toBe(600_000);
  });
});

describe('drainQueue', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  async function enqueuePerson(syncId: string, name: string) {
    await db.persons.put({
      id: syncId,
      data: { id: syncId, full_name: name },
      syncStatus: 'pending',
      localCreatedAt: new Date().toISOString(),
    });
    await db.syncQueue.add({
      entityType: 'person',
      entityId: syncId,
      action: 'create',
      data: { sync_id: syncId, full_name: name },
      createdAt: new Date().toISOString(),
      retryCount: 0,
    });
  }

  it('returns idle result when the queue is empty', async () => {
    const push: PushFn = vi.fn();
    const result = await drainQueue(push);
    expect(result.pushed).toBe(0);
    expect(push).not.toHaveBeenCalled();
  });

  it('pushes queued records and removes them on created status', async () => {
    await enqueuePerson('p1', 'Maria');
    await enqueuePerson('p2', 'João');

    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({
        sync_id: r.sync_id,
        status: 'created' as const,
        server_id: 'srv-' + r.sync_id,
      })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);

    expect(push).toHaveBeenCalledOnce();
    expect(result.pushed).toBe(2);
    expect(result.conflicts).toBe(0);
    expect(await db.syncQueue.count()).toBe(0);
    // Cached person flips to synced.
    expect((await db.persons.get('p1'))?.syncStatus).toBe('synced');
  });

  it('flags conflict WITHOUT dropping the data, and excludes it from later drains', async () => {
    await enqueuePerson('p3', 'Ana');

    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({
        sync_id: r.sync_id,
        status: 'conflict' as const,
        message: 'duplicate',
      })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);

    expect(result.conflicts).toBe(1);
    expect((await db.persons.get('p3'))?.syncStatus).toBe('conflict');
    // The field-captured record is NOT lost: the queue item is preserved
    // (flagged conflicted) so it can be reviewed/resubmitted, not deleted.
    const items = await db.syncQueue.toArray();
    expect(items).toHaveLength(1);
    expect(items[0]?.conflicted).toBe(true);

    // A second drain must NOT re-push the conflicted item (no thrash, no dup).
    const push2: PushFn = vi.fn();
    const again = await drainQueue(push2);
    expect(push2).not.toHaveBeenCalled();
    expect(again.pushed).toBe(0);
  });

  it('keeps the item and increments retryCount on an error-status result', async () => {
    await enqueuePerson('p5', 'Lia');

    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({
        sync_id: r.sync_id,
        status: 'error' as const,
        message: 'transient server error',
      })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);

    expect(result.failed).toBe(1);
    expect(result.pushed).toBe(0);
    const items = await db.syncQueue.toArray();
    expect(items).toHaveLength(1);
    expect(items[0]?.retryCount).toBe(1);
    expect(items[0]?.lastError).toContain('transient server error');
    // The cached record is left pending (not synced, not conflict).
    expect((await db.persons.get('p5'))?.syncStatus).toBe('pending');
  });

  it('ignores a result whose sync_id is not in the batch', async () => {
    await enqueuePerson('p6', 'Rui');

    const push: PushFn = vi.fn(async () => ({
      results: [{ sync_id: 'unknown-id', status: 'created' as const, server_id: 'x' }],
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);

    expect(result.pushed).toBe(0);
    // The real item is untouched (still queued) since its result was absent.
    expect(await db.syncQueue.count()).toBe(1);
  });

  it('dead-letters an item once it exceeds MAX_RETRIES (no infinite retry)', async () => {
    await enqueuePerson('p7', 'Sol');
    // Pre-age the item to one attempt below the limit.
    const item = (await db.syncQueue.toArray())[0]!;
    await db.syncQueue.update(item.id!, { retryCount: MAX_RETRIES });

    const push: PushFn = vi.fn(async () => {
      throw new Error('still down');
    });

    const result = await drainQueue(push);
    expect(result.failed).toBe(1);

    // The item is now dead-lettered: a further drain skips it (no poison loop).
    const push2: PushFn = vi.fn();
    await drainQueue(push2);
    expect(push2).not.toHaveBeenCalled();
    const dead = (await db.syncQueue.toArray())[0]!;
    expect(dead.deadLettered).toBe(true);
  });

  it('increments retryCount and keeps the item when the push throws', async () => {
    await enqueuePerson('p4', 'Pedro');

    const push: PushFn = vi.fn(async () => {
      throw new Error('network down');
    });

    const result = await drainQueue(push);

    expect(result.pushed).toBe(0);
    expect(result.failed).toBe(1);
    const items = await db.syncQueue.toArray();
    expect(items).toHaveLength(1);
    expect(items[0]?.retryCount).toBe(1);
    expect(items[0]?.lastError).toContain('network down');
  });
});

describe('conflict / retry queue helpers', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  async function queue(entityId: string, patch: Record<string, unknown> = {}) {
    return db.syncQueue.add({
      entityType: 'person',
      entityId,
      action: 'create',
      data: { sync_id: entityId },
      createdAt: new Date().toISOString(),
      retryCount: 0,
      ...patch,
    });
  }

  it('getConflicts returns only conflicted items', async () => {
    await queue('a');
    await queue('b', { conflicted: true });
    const conflicts = await getConflicts();
    expect(conflicts.map((c) => c.entityId)).toEqual(['b']);
  });

  it('getRetryable excludes conflicted and dead-lettered items', async () => {
    await queue('a');
    await queue('b', { conflicted: true });
    await queue('c', { deadLettered: true });
    const retryable = await getRetryable();
    expect(retryable.map((r) => r.entityId)).toEqual(['a']);
  });

  it('discardConflict removes a reviewed conflicted item', async () => {
    const key = (await queue('b', { conflicted: true })) as number;
    await discardConflict(key);
    expect(await db.syncQueue.count()).toBe(0);
  });

  it('requeueConflict clears the conflicted flag and makes the item retryable', async () => {
    const key = (await queue('b', {
      conflicted: true,
      lastError: 'server conflict',
      retryCount: 2,
    })) as number;
    await db.persons.put({
      id: 'b',
      data: { id: 'b', full_name: 'Preserved' },
      syncStatus: 'conflict',
      localCreatedAt: 'x',
    });

    await requeueConflict(key);

    const item = await db.syncQueue.get(key);
    expect(item?.conflicted).toBeFalsy();
    // A resubmit is a fresh attempt: the retry counter resets so a previously
    // error-bumped item does not start near the dead-letter ceiling.
    expect(item?.retryCount).toBe(0);
    // The cached entity returns to pending so it is not shown as a conflict.
    expect((await db.persons.get('b'))?.syncStatus).toBe('pending');
    // It is now eligible for the next automatic drain again.
    expect((await getRetryable()).map((r) => r.entityId)).toContain('b');
    expect((await getConflicts()).length).toBe(0);
  });

  it('resolves a created result even when no cached entity row exists', async () => {
    // Queue item without a corresponding cached person (e.g. cache evicted):
    // the drain must still clear the queue item without throwing.
    await queue('orphan');
    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({
        sync_id: r.sync_id,
        status: 'created' as const,
        server_id: 'srv',
      })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);
    expect(result.pushed).toBe(1);
    expect(await db.syncQueue.count()).toBe(0);
    expect(await db.persons.get('orphan')).toBeUndefined();
  });

  it('flags a conflict even when no cached entity row exists', async () => {
    await queue('orphan2');
    const push: PushFn = vi.fn(async (records: SyncPushRecord[]) => ({
      results: records.map((r) => ({ sync_id: r.sync_id, status: 'conflict' as const })),
      server_timestamp: '2026-01-01T00:00:00Z',
    }));

    const result = await drainQueue(push);
    expect(result.conflicts).toBe(1);
    const items = await db.syncQueue.toArray();
    expect(items[0]?.conflicted).toBe(true);
  });
});
