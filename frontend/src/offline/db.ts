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

/**
 * ConflictSnapshot holds the server's version of a record that conflicted with a
 * local change (S12.2). It is captured on both the push path (when the server
 * returns `server_data`) and the pull path (the server record that previously
 * would have been discarded), so the field-level resolution UI can present the
 * server value even while offline. `serverData` is a lean, non-PII projection —
 * the same field set the client already holds — never the full server row.
 */
export interface ConflictSnapshot {
  entityId: string;
  entityType: SyncEntityType;
  serverData: Record<string, unknown>;
  serverUpdatedAt?: string;
  conflictFields?: string[];
  capturedAt: string;
}

class ChesedDB extends Dexie {
  persons!: Table<LocalEntity>;
  triages!: Table<LocalEntity>;
  attendances!: Table<LocalEntity>;
  syncQueue!: Table<SyncQueueItem>;
  syncMeta!: Table<SyncMeta>;
  conflicts!: Table<ConflictSnapshot>;

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
    // v3 — add the conflicts store for field-level resolution (S12.2). Additive:
    // no existing store is altered, so the upgrade preserves cached data and the
    // queue. Keyed by entityId (one snapshot per conflicted record), indexed by
    // entityType for per-type listing.
    this.version(3).stores({
      conflicts: 'entityId, entityType',
    });
  }
}

export const db = new ChesedDB();
