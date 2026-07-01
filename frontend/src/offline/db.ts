import Dexie, { type Table } from 'dexie';

/**
 * LocalEntity is the generic shape for any offline-cached record (person,
 * triage, attendance). `data` holds the full server/client payload; the sync
 * fields track its reconciliation state.
 */
export interface LocalEntity {
  id: string;
  data: Record<string, unknown>;
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

/** LocalPerson is kept as an alias for backward compatibility. */
export type LocalPerson = LocalEntity;

export type SyncEntityType = 'person' | 'triage' | 'attendance';

export interface SyncQueueItem {
  id?: number;
  entityType: SyncEntityType;
  entityId: string;
  action: 'create' | 'update';
  data: Record<string, unknown>;
  createdAt: string;
  retryCount: number;
  lastError?: string;
  /**
   * Set when the server rejected the push as a conflict. Conflicted items are
   * preserved (never auto-dropped) and excluded from further drains until an
   * operator reviews/resubmits or discards them, so field data is not lost.
   */
  conflicted?: boolean;
  /**
   * Set when the item exceeded the retry ceiling. Dead-lettered items are kept
   * (never lost) but excluded from auto-drains so a persistently-failing record
   * cannot poison the queue; an operator reviews/resubmits them.
   */
  deadLettered?: boolean;
}

export interface SyncMeta {
  key: string;
  value: string;
}

class ChesedDB extends Dexie {
  persons!: Table<LocalEntity>;
  triages!: Table<LocalEntity>;
  attendances!: Table<LocalEntity>;
  syncQueue!: Table<SyncQueueItem>;
  syncMeta!: Table<SyncMeta>;

  constructor() {
    super('chesed-offline');
    // v1 — person cache + sync queue (shipped in the Sprint 4 backend slice).
    this.version(1).stores({
      persons: 'id, syncStatus',
      syncQueue: '++id, entityType, entityId, createdAt',
      syncMeta: 'key',
    });
    // v2 — add triage and attendance offline stores for the sync drainer.
    // Additive change: existing stores are unchanged, so the upgrade is
    // non-destructive and cached persons / queued items are preserved.
    this.version(2).stores({
      triages: 'id, syncStatus',
      attendances: 'id, syncStatus',
    });
  }
}

export const db = new ChesedDB();
