import { describe, expect, it, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';
import { mergePullRecords, type PullRecord } from '../syncEngine';

describe('mergePullRecords', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  function record(id: string, updatedAt: string, name: string): PullRecord {
    return {
      entity_type: 'person',
      id,
      data: { id, full_name: name },
      updated_at: updatedAt,
    };
  }

  it('inserts a record that is not cached locally', async () => {
    const merged = await mergePullRecords([record('p1', '2026-01-02T00:00:00Z', 'Maria')]);
    expect(merged.applied).toBe(1);
    const cached = await db.persons.get('p1');
    expect(cached?.data.full_name).toBe('Maria');
    expect(cached?.syncStatus).toBe('synced');
    expect(cached?.serverUpdatedAt).toBe('2026-01-02T00:00:00Z');
  });

  it('replaces a local record when the server copy is newer', async () => {
    await db.persons.put({
      id: 'p2',
      data: { id: 'p2', full_name: 'Old Name' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-01T00:00:00Z',
      serverUpdatedAt: '2026-01-01T00:00:00Z',
    });

    const merged = await mergePullRecords([record('p2', '2026-01-03T00:00:00Z', 'New Name')]);

    expect(merged.applied).toBe(1);
    expect((await db.persons.get('p2'))?.data.full_name).toBe('New Name');
  });

  it('keeps the local record when it is not older than the server copy', async () => {
    await db.persons.put({
      id: 'p3',
      data: { id: 'p3', full_name: 'Local Wins' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-05T00:00:00Z',
      serverUpdatedAt: '2026-01-05T00:00:00Z',
    });

    const merged = await mergePullRecords([record('p3', '2026-01-04T00:00:00Z', 'Stale Server')]);

    expect(merged.skipped).toBe(1);
    expect((await db.persons.get('p3'))?.data.full_name).toBe('Local Wins');
  });

  it('does NOT overwrite a record with a pending local mutation (flags conflict)', async () => {
    await db.persons.put({
      id: 'p4',
      data: { id: 'p4', full_name: 'My Offline Edit' },
      syncStatus: 'pending',
      localCreatedAt: '2026-01-01T00:00:00Z',
    });

    const merged = await mergePullRecords([record('p4', '2026-01-09T00:00:00Z', 'Server Edit')]);

    expect(merged.conflicts).toBe(1);
    const cached = await db.persons.get('p4');
    expect(cached?.data.full_name).toBe('My Offline Edit'); // local data preserved
    expect(cached?.syncStatus).toBe('conflict');
  });

  it('routes triage and attendance records to their own stores', async () => {
    await mergePullRecords([
      { entity_type: 'triage', id: 't1', data: { id: 't1', main_complaint: 'x' }, updated_at: '2026-01-02T00:00:00Z' },
      { entity_type: 'attendance', id: 'a1', data: { id: 'a1', status: 'SCHEDULED' }, updated_at: '2026-01-02T00:00:00Z' },
    ]);
    expect(await db.triages.get('t1')).toBeDefined();
    expect(await db.attendances.get('a1')).toBeDefined();
  });
});
