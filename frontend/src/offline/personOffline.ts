import { db, type LocalPerson } from './db';
import type { PersonListItem, CreatePersonInput } from '../types';
import { createPerson } from '../api/persons';
import { isNetworkError } from '../api/errors';

export interface OfflineCreateResult {
  id: string;
  offline: boolean;
}

/**
 * createPersonWithOfflineFallback creates a person via the API when online, and
 * transparently queues it to IndexedDB when offline or when the request fails
 * with a network error. Application errors (validation, conflict) are re-thrown.
 */
export async function createPersonWithOfflineFallback(
  input: CreatePersonInput,
): Promise<OfflineCreateResult> {
  if (!navigator.onLine) {
    return { id: await savePersonOffline(input), offline: true };
  }
  try {
    const created = await createPerson(input);
    return { id: created.id, offline: false };
  } catch (err) {
    if (isNetworkError(err)) {
      return { id: await savePersonOffline(input), offline: true };
    }
    throw err;
  }
}

export async function cachePersonList(persons: PersonListItem[]): Promise<void> {
  const items: LocalPerson[] = persons.map((p) => ({
    id: p.id,
    data: p as unknown as Record<string, unknown>,
    syncStatus: 'synced' as const,
    localCreatedAt: new Date().toISOString(),
  }));

  await db.persons.bulkPut(items);
}

export async function getCachedPersons(): Promise<PersonListItem[]> {
  const items = await db.persons.toArray();
  return items.map((item) => item.data as unknown as PersonListItem);
}

export async function savePersonOffline(
  input: CreatePersonInput,
): Promise<string> {
  const id = crypto.randomUUID();

  // Build a valid PersonListItem so the list/card components render the cached
  // record offline without crashing on fields the create input does not carry
  // (roles, is_active). The server fills these in once the record syncs.
  const listItem: PersonListItem = {
    id,
    full_name: input.full_name,
    document_number: input.document_number ?? undefined,
    phone: input.phone ?? undefined,
    roles: [],
    is_active: true,
  };

  await db.persons.put({
    id,
    data: listItem as unknown as Record<string, unknown>,
    syncStatus: 'pending',
    localCreatedAt: new Date().toISOString(),
  });

  await db.syncQueue.add({
    entityType: 'person',
    entityId: id,
    action: 'create',
    data: { ...input, sync_id: id } as unknown as Record<string, unknown>,
    createdAt: new Date().toISOString(),
    retryCount: 0,
  });

  return id;
}

export async function getPendingSyncCount(): Promise<number> {
  return db.syncQueue.count();
}

export async function getOfflinePerson(
  id: string,
): Promise<LocalPerson | undefined> {
  return db.persons.get(id);
}
