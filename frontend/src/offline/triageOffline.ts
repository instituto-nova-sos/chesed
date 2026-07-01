import { db, type LocalEntity } from './db';
import type { TriageListItem, CreateTriageInput } from '../types';
import { createTriage } from '../api/triages';
import { isNetworkError } from '../api/errors';

export interface OfflineCreateResult {
  id: string;
  offline: boolean;
}

/**
 * createTriageWithOfflineFallback creates a triage via the API when online, and
 * transparently queues it to IndexedDB when offline or when the request fails
 * with a network error. Application errors (validation, conflict) are re-thrown.
 * `personName` is stored on the cached list item so the offline list renders a
 * readable row before the record syncs.
 */
export async function createTriageWithOfflineFallback(
  input: CreateTriageInput,
  personName?: string,
): Promise<OfflineCreateResult> {
  if (!navigator.onLine) {
    return { id: await saveTriageOffline(input, personName), offline: true };
  }
  try {
    const created = await createTriage(input);
    return { id: created.id, offline: false };
  } catch (err) {
    if (isNetworkError(err)) {
      return { id: await saveTriageOffline(input, personName), offline: true };
    }
    throw err;
  }
}

export async function cacheTriageList(triages: TriageListItem[]): Promise<void> {
  const items: LocalEntity[] = triages.map((t) => ({
    id: t.id,
    data: t as unknown as Record<string, unknown>,
    syncStatus: 'synced' as const,
    localCreatedAt: new Date().toISOString(),
  }));

  await db.triages.bulkPut(items);
}

export async function getCachedTriages(): Promise<TriageListItem[]> {
  const items = await db.triages.toArray();
  return items.map((item) => item.data as unknown as TriageListItem);
}

export async function saveTriageOffline(
  input: CreateTriageInput,
  personName?: string,
): Promise<string> {
  const id = crypto.randomUUID();

  // Build a valid TriageListItem so the list renders the cached record offline
  // without crashing on fields the create input does not carry (person_name,
  // requested_service_count). The server fills these in once the record syncs.
  const listItem: TriageListItem = {
    id,
    person_id: input.person_id,
    person_name: personName ?? '',
    main_complaint: input.main_complaint,
    triage_date: input.triage_date ?? new Date().toISOString(),
    requested_service_count: input.requested_service_types?.length ?? 0,
  };

  await db.triages.put({
    id,
    data: listItem as unknown as Record<string, unknown>,
    syncStatus: 'pending',
    localCreatedAt: new Date().toISOString(),
  });

  await db.syncQueue.add({
    entityType: 'triage',
    entityId: id,
    action: 'create',
    data: { ...input, sync_id: id } as unknown as Record<string, unknown>,
    createdAt: new Date().toISOString(),
    retryCount: 0,
  });

  return id;
}
