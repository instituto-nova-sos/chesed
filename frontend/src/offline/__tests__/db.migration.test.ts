import { describe, expect, it, afterEach } from 'vitest';
import 'fake-indexeddb/auto';
import Dexie from 'dexie';

// S05.1 criteria 2 & 4: the v1→v2 upgrade must preserve existing offline records
// (no data loss), and records must survive a close/reopen cycle (reload durability).
// This suite manages its own DB lifecycle because it opens a bare v1 handle against
// the same backing store as the real v2 singleton.
const DB_NAME = 'chesed-offline';

function openV1(): Dexie {
  const v1 = new Dexie(DB_NAME);
  v1.version(1).stores({
    persons: 'id, syncStatus',
    syncQueue: '++id, entityType, entityId, createdAt',
    syncMeta: 'key',
  });
  return v1;
}

describe('offline db v1 → v2 migration', () => {
  afterEach(async () => {
    await Dexie.delete(DB_NAME);
  });

  it('preserves v1 records after upgrading to the v2 schema', async () => {
    const v1 = openV1();
    await v1.open();
    await v1.table('persons').put({
      id: 'p-v1',
      data: { full_name: 'Legacy Person' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-01T00:00:00.000Z',
    });
    await v1.table('syncQueue').add({
      entityType: 'person',
      entityId: 'p-v1',
      action: 'create',
      data: { sync_id: 'p-v1' },
      createdAt: '2026-01-01T00:00:00.000Z',
      retryCount: 0,
    });
    v1.close();

    // Import the real v2 singleton only after the v1 store exists, so the version
    // transition runs against populated data (the actual migration path).
    const { db } = await import('../db');
    await db.open();

    expect(db.verno).toBe(2);
    const person = await db.persons.get('p-v1');
    expect(person?.data.full_name).toBe('Legacy Person');
    const queued = await db.syncQueue.toArray();
    expect(queued).toHaveLength(1);
    expect(queued[0]?.entityId).toBe('p-v1');

    // The new v2 stores must be usable after the upgrade.
    await db.triages.put({
      id: 't-1',
      data: { main_complaint: 'x' },
      syncStatus: 'pending',
      localCreatedAt: '2026-01-01T00:00:00.000Z',
    });
    expect(await db.triages.get('t-1')).toBeTruthy();
    db.close();
  });

  it('keeps records durable across a close/reopen cycle', async () => {
    const { db } = await import('../db');
    await db.open();
    await db.persons.put({
      id: 'p-durable',
      data: { full_name: 'Durable' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-01T00:00:00.000Z',
    });
    db.close();

    await db.open();
    const reread = await db.persons.get('p-durable');
    expect(reread?.data.full_name).toBe('Durable');
    db.close();
  });
});
