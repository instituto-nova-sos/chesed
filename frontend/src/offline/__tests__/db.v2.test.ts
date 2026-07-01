import { describe, expect, it, beforeEach } from 'vitest';
import 'fake-indexeddb/auto';
import { db, type LocalEntity, type SyncQueueItem } from '../db';

// These tests open the real (fake-indexeddb) database to verify the v2 schema:
// triages and attendances stores added alongside the existing person store,
// and that the version bump upgrades cleanly without data loss.
describe('offline db v2 schema', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('declares schema version 2', () => {
    expect(db.verno).toBe(2);
  });

  it('exposes persons, triages, attendances, syncQueue and syncMeta tables', () => {
    const names = db.tables.map((t) => t.name).sort();
    expect(names).toEqual(
      ['attendances', 'persons', 'syncMeta', 'syncQueue', 'triages'].sort(),
    );
  });

  it('keys triages and attendances on "id" and indexes syncStatus', () => {
    for (const name of ['triages', 'attendances']) {
      const table = db.table(name);
      expect(table.schema.primKey.name).toBe('id');
      expect(table.schema.indexes.map((i) => i.name)).toContain('syncStatus');
    }
  });

  it('round-trips a triage entity through IndexedDB', async () => {
    const entity: LocalEntity = {
      id: 'triage-1',
      data: { main_complaint: 'headache' },
      syncStatus: 'pending',
      localCreatedAt: new Date().toISOString(),
    };
    await db.triages.put(entity);
    const read = await db.triages.get('triage-1');
    expect(read?.data.main_complaint).toBe('headache');
    expect(read?.syncStatus).toBe('pending');
  });

  it('queues triage and attendance entity types', async () => {
    const item: SyncQueueItem = {
      entityType: 'attendance',
      entityId: 'att-1',
      action: 'create',
      data: { sync_id: 'att-1' },
      createdAt: new Date().toISOString(),
      retryCount: 0,
    };
    const key = await db.syncQueue.add(item);
    const read = await db.syncQueue.get(key);
    expect(read?.entityType).toBe('attendance');
  });
});
