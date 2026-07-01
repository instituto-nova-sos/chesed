import { db, type LocalEntity, type SyncEntityType, type SyncQueueItem } from './db';

/** A record as sent in a sync push batch. */
export interface SyncPushRecord {
  entity_type: SyncEntityType;
  sync_id: string;
  data: Record<string, unknown>;
}

export interface SyncPushResult {
  sync_id: string;
  status: 'created' | 'conflict' | 'error';
  server_id?: string;
  message?: string;
}

export interface SyncPushResponse {
  results: SyncPushResult[];
  server_timestamp: string;
}

/** PushFn uploads a batch of records and returns the per-record results. */
export type PushFn = (records: SyncPushRecord[]) => Promise<SyncPushResponse>;

export interface DrainResult {
  pushed: number;
  conflicts: number;
  failed: number;
}

/** Maximum records per push batch (mirrors the backend 50-record cap). */
export const MAX_BATCH = 50;

/** Retry ceiling before an item is dead-lettered for operator review. */
export const MAX_RETRIES = 5;

const BACKOFF_SCHEDULE_MS = [5_000, 30_000, 120_000, 600_000];

/**
 * backoffDelay returns the retry delay for a given attempt index, following the
 * 5s → 30s → 2m → 10m schedule and capping at 10m for all further attempts.
 */
export function backoffDelay(attempt: number): number {
  const idx = Math.min(Math.max(attempt, 0), BACKOFF_SCHEDULE_MS.length - 1);
  return BACKOFF_SCHEDULE_MS[idx] ?? BACKOFF_SCHEDULE_MS[BACKOFF_SCHEDULE_MS.length - 1] ?? 0;
}

const TABLE_BY_TYPE: Record<SyncEntityType, 'persons' | 'triages' | 'attendances'> = {
  person: 'persons',
  triage: 'triages',
  attendance: 'attendances',
};

function toPushRecord(item: SyncQueueItem): SyncPushRecord {
  return {
    entity_type: item.entityType,
    sync_id: item.entityId,
    data: item.data,
  };
}

/**
 * drainQueue pushes all currently-queued records in a single batch and applies
 * the per-record results:
 *   - created  → mark the cached entity 'synced', remove the queue item
 *   - conflict → mark the cached entity 'conflict', FLAG the queue item
 *                `conflicted` (preserved, never dropped) so the captured data is
 *                recoverable; conflicted items are excluded from later drains
 *   - error    → increment retryCount and keep the queue item for a later
 *                attempt, dead-lettering it once it exceeds MAX_RETRIES
 * A thrown push (offline / network) increments retryCount on every item and
 * keeps them queued (same dead-letter ceiling applies). Conflicted and
 * dead-lettered items are skipped on subsequent drains.
 */
export async function drainQueue(push: PushFn): Promise<DrainResult> {
  // Conflicted and dead-lettered items are excluded: they await operator
  // resolution and must not be re-pushed (which would thrash or duplicate).
  const queued = (await db.syncQueue.orderBy('createdAt').toArray())
    .filter((item) => !item.conflicted && !item.deadLettered)
    .slice(0, MAX_BATCH);
  if (queued.length === 0) {
    return { pushed: 0, conflicts: 0, failed: 0 };
  }

  let response: SyncPushResponse;
  try {
    response = await push(queued.map(toPushRecord));
  } catch (err) {
    await markBatchFailed(queued, err);
    return { pushed: 0, conflicts: 0, failed: queued.length };
  }

  return applyResults(queued, response.results);
}

async function markBatchFailed(items: SyncQueueItem[], err: unknown): Promise<void> {
  const message = err instanceof Error ? err.message : String(err);
  await db.transaction('rw', db.syncQueue, async () => {
    for (const item of items) {
      await bumpRetry(item, message);
    }
  });
}

async function applyResults(
  items: SyncQueueItem[],
  results: SyncPushResult[],
): Promise<DrainResult> {
  const byID = new Map(items.map((i) => [i.entityId, i]));
  let pushed = 0;
  let conflicts = 0;
  let failed = 0;

  for (const result of results) {
    const item = byID.get(result.sync_id);
    if (!item) continue;

    if (result.status === 'created') {
      await resolveCreated(item);
      pushed += 1;
    } else if (result.status === 'conflict') {
      await flagConflict(item, result.message ?? 'conflict');
      conflicts += 1;
    } else {
      await bumpRetry(item, result.message ?? 'sync error');
      failed += 1;
    }
  }

  return { pushed, conflicts, failed };
}

/** resolveCreated marks the cached entity synced and removes the queue item. */
async function resolveCreated(item: SyncQueueItem): Promise<void> {
  await setCachedStatus(item, 'synced');
  if (item.id !== undefined) {
    await db.syncQueue.delete(item.id);
  }
}

/**
 * flagConflict marks the cached entity 'conflict' and flags the queue item
 * conflicted WITHOUT deleting it, so the field-captured record is preserved for
 * operator review/resubmission. The item is excluded from further auto-drains.
 */
async function flagConflict(item: SyncQueueItem, message: string): Promise<void> {
  await setCachedStatus(item, 'conflict');
  if (item.id !== undefined) {
    await db.syncQueue.update(item.id, { conflicted: true, lastError: message });
  }
}

async function setCachedStatus(
  item: SyncQueueItem,
  status: 'synced' | 'conflict',
): Promise<void> {
  const table = db.table(TABLE_BY_TYPE[item.entityType]);
  const cached = await table.get(item.entityId);
  if (cached) {
    await table.update(item.entityId, { syncStatus: status });
  }
}

/** getConflicts returns the queued items awaiting operator conflict resolution. */
export async function getConflicts(): Promise<SyncQueueItem[]> {
  return (await db.syncQueue.toArray()).filter((item) => item.conflicted);
}

/** getRetryable returns queued items still eligible for an automatic retry. */
export async function getRetryable(): Promise<SyncQueueItem[]> {
  return (await db.syncQueue.toArray()).filter(
    (item) => !item.conflicted && !item.deadLettered,
  );
}

/** discardConflict permanently removes a conflicted queue item after review. */
export async function discardConflict(queueId: number): Promise<void> {
  await db.syncQueue.delete(queueId);
}

/**
 * requeueConflict clears the `conflicted` flag on a queue item and returns its
 * cached entity to `pending`, so the next drain re-pushes it (last-write-wins
 * resubmission after an operator reviews the conflict). The captured data is
 * unchanged — only its eligibility for auto-drain is restored.
 */
export async function requeueConflict(queueId: number): Promise<void> {
  const item = await db.syncQueue.get(queueId);
  if (!item) return;
  // A resubmit is a fresh attempt: reset retryCount so an item that hit several
  // transient errors before the conflict does not start near the dead-letter
  // ceiling on its next drain.
  await db.syncQueue.update(queueId, {
    conflicted: false,
    lastError: undefined,
    retryCount: 0,
  });
  const table = db.table<LocalEntity>(TABLE_BY_TYPE[item.entityType]);
  const cached = await table.get(item.entityId);
  if (cached) {
    await table.update(item.entityId, { syncStatus: 'pending' });
  }
}

async function bumpRetry(item: SyncQueueItem, message: string): Promise<void> {
  if (item.id === undefined) return;
  const nextRetry = item.retryCount + 1;
  await db.syncQueue.update(item.id, {
    retryCount: nextRetry,
    lastError: message,
    // Dead-letter once the ceiling is reached so a persistently-failing record
    // cannot loop forever; it is preserved for operator review, not dropped.
    ...(nextRetry > MAX_RETRIES ? { deadLettered: true } : {}),
  });
}

/** A record returned by a sync pull. */
export interface PullRecord {
  entity_type: SyncEntityType;
  id: string;
  data: Record<string, unknown>;
  updated_at: string;
}

export interface MergeResult {
  applied: number;
  skipped: number;
  conflicts: number;
}

/**
 * mergePullRecords reconciles server records into the local cache:
 *   - absent locally           → insert as 'synced'
 *   - local has pending edit    → flag 'conflict', preserve local data
 *   - server strictly newer     → replace, mark 'synced'
 *   - otherwise (local newer/=)  → skip
 */
export async function mergePullRecords(records: PullRecord[]): Promise<MergeResult> {
  let applied = 0;
  let skipped = 0;
  let conflicts = 0;

  for (const record of records) {
    const table = db.table<LocalEntity>(TABLE_BY_TYPE[record.entity_type]);
    const local = await table.get(record.id);

    if (!local) {
      await table.put(toSyncedEntity(record, local));
      applied += 1;
      continue;
    }
    if (local.syncStatus === 'pending') {
      await table.update(record.id, { syncStatus: 'conflict' });
      conflicts += 1;
      continue;
    }
    if (isServerNewer(record, local)) {
      await table.put(toSyncedEntity(record, local));
      applied += 1;
    } else {
      skipped += 1;
    }
  }

  return { applied, skipped, conflicts };
}

function toSyncedEntity(record: PullRecord, prev: LocalEntity | undefined): LocalEntity {
  return {
    id: record.id,
    data: record.data,
    syncStatus: 'synced',
    localCreatedAt: prev?.localCreatedAt ?? new Date().toISOString(),
    serverUpdatedAt: record.updated_at,
  };
}

function isServerNewer(record: PullRecord, local: LocalEntity): boolean {
  const localStamp = local.serverUpdatedAt ?? local.localCreatedAt;
  return new Date(record.updated_at).getTime() > new Date(localStamp).getTime();
}
