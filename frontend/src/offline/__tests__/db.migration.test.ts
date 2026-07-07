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

function openV2(): Dexie {
  const v2 = new Dexie(DB_NAME);
  v2.version(1).stores({
    persons: 'id, syncStatus',
    syncQueue: '++id, entityType, entityId, createdAt',
    syncMeta: 'key',
  });
  v2.version(2).stores({
    triages: 'id, syncStatus',
    attendances: 'id, syncStatus',
  });
  return v2;
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

    // Import the real singleton only after the v1 store exists, so the version
    // transition runs against populated data (the actual migration path). The
    // singleton upgrades to its highest declared version (currently v3), applying
    // every intermediate upgrade — this asserts v1 data survives that full chain.
    const { db } = await import('../db');
    await db.open();

    expect(db.verno).toBe(3);
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

// S12.2: the v2→v3 upgrade adds the conflicts store for field-level conflict
// resolution. The change is additive, so existing person / triage / attendance
// records and the sync queue must survive the upgrade with no data loss.
describe('offline db v2 → v3 migration', () => {
  afterEach(async () => {
    await Dexie.delete(DB_NAME);
  });

  it('preserves v2 records after upgrading to the v3 schema', async () => {
    const v2 = openV2();
    await v2.open();
    await v2.table('persons').put({
      id: 'p-v2',
      data: { full_name: 'V2 Person' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-01T00:00:00.000Z',
    });
    await v2.table('triages').put({
      id: 't-v2',
      data: { main_complaint: 'legacy triage' },
      syncStatus: 'synced',
      localCreatedAt: '2026-01-01T00:00:00.000Z',
    });
    await v2.table('syncQueue').add({
      entityType: 'person',
      entityId: 'p-v2',
      action: 'create',
      data: { sync_id: 'p-v2' },
      createdAt: '2026-01-01T00:00:00.000Z',
      retryCount: 0,
    });
    v2.close();

    // Import the real v3 singleton only after the v2 store exists, so the version
    // transition runs against populated data (the actual migration path).
    const { db } = await import('../db');
    await db.open();

    expect(db.verno).toBe(3);
    expect((await db.persons.get('p-v2'))?.data.full_name).toBe('V2 Person');
    expect((await db.triages.get('t-v2'))?.data.main_complaint).toBe('legacy triage');
    const queued = await db.syncQueue.toArray();
    expect(queued).toHaveLength(1);
    expect(queued[0]?.entityId).toBe('p-v2');

    // The new v3 conflicts store must be usable after the upgrade.
    await db.conflicts.put({
      entityId: 'p-v2',
      entityType: 'person',
      serverData: { full_name: 'Server version' },
      capturedAt: '2026-01-02T00:00:00.000Z',
    });
    expect((await db.conflicts.get('p-v2'))?.serverData.full_name).toBe('Server version');
    db.close();
  });
});
