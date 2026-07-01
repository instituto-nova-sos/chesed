import { db, type LocalEntity } from './db';
import type { AttendanceListItem, CreateAttendanceInput } from '../types';
import { createAttendance } from '../api/attendances';
import { isNetworkError } from '../api/errors';

export interface OfflineCreateResult {
  id: string;
  offline: boolean;
}

/**
 * createAttendanceWithOfflineFallback creates an attendance via the API when
 * online, and transparently queues it to IndexedDB when offline or when the
 * request fails with a network error. Application errors (validation, conflict)
 * are re-thrown. `personName`/`serviceType` are stored on the cached list item
 * so the offline list renders a readable row before the record syncs.
 */
export async function createAttendanceWithOfflineFallback(
  input: CreateAttendanceInput,
  personName?: string,
  serviceType?: string,
): Promise<OfflineCreateResult> {
  if (!navigator.onLine) {
    return { id: await saveAttendanceOffline(input, personName, serviceType), offline: true };
  }
  try {
    const created = await createAttendance(input);
    return { id: created.id, offline: false };
  } catch (err) {
    if (isNetworkError(err)) {
      return { id: await saveAttendanceOffline(input, personName, serviceType), offline: true };
    }
    throw err;
  }
}

export async function cacheAttendanceList(attendances: AttendanceListItem[]): Promise<void> {
  const items: LocalEntity[] = attendances.map((a) => ({
    id: a.id,
    data: a as unknown as Record<string, unknown>,
    syncStatus: 'synced' as const,
    localCreatedAt: new Date().toISOString(),
  }));

  await db.attendances.bulkPut(items);
}

export async function getCachedAttendances(): Promise<AttendanceListItem[]> {
  const items = await db.attendances.toArray();
  return items.map((item) => item.data as unknown as AttendanceListItem);
}

export async function saveAttendanceOffline(
  input: CreateAttendanceInput,
  personName?: string,
  serviceType?: string,
): Promise<string> {
  const id = crypto.randomUUID();

  // Build a valid AttendanceListItem so the list renders the cached record
  // offline without crashing on fields the create input does not carry
  // (person_name, service_type, status). A new attendance defaults to SCHEDULED
  // per the Phase 1 state machine; the server confirms it once the record syncs.
  const listItem: AttendanceListItem = {
    id,
    person_id: input.person_id,
    person_name: personName ?? '',
    service_type: serviceType ?? '',
    status: 'SCHEDULED',
    attendance_date: input.attendance_date ?? new Date().toISOString(),
  };

  await db.attendances.put({
    id,
    data: listItem as unknown as Record<string, unknown>,
    syncStatus: 'pending',
    localCreatedAt: new Date().toISOString(),
  });

  await db.syncQueue.add({
    entityType: 'attendance',
    entityId: id,
    action: 'create',
    data: { ...input, sync_id: id } as unknown as Record<string, unknown>,
    createdAt: new Date().toISOString(),
    retryCount: 0,
  });

  return id;
}
