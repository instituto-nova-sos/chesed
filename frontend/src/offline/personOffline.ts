import { db, type LocalPerson } from './db';
import type { PersonListItem, CreatePersonInput } from '../types';

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

  await db.persons.put({
    id,
    data: { ...input, id } as unknown as Record<string, unknown>,
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
