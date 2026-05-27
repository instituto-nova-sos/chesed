import Dexie, { type Table } from 'dexie';

export interface LocalPerson {
  id: string;
  data: Record<string, unknown>;
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

export interface SyncQueueItem {
  id?: number;
  entityType: 'person' | 'triage' | 'attendance';
  entityId: string;
  action: 'create' | 'update';
  data: Record<string, unknown>;
  createdAt: string;
  retryCount: number;
  lastError?: string;
}

export interface SyncMeta {
  key: string;
  value: string;
}

class ChesedDB extends Dexie {
  persons!: Table<LocalPerson>;
  syncQueue!: Table<SyncQueueItem>;
  syncMeta!: Table<SyncMeta>;

  constructor() {
    super('chesed-offline');
    this.version(1).stores({
      persons: 'id, syncStatus',
      syncQueue: '++id, entityType, entityId, createdAt',
      syncMeta: 'key',
    });
  }
}

export const db = new ChesedDB();
