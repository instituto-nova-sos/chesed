import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';

const createAttendance = vi.fn();
vi.mock('../../api/attendances', () => ({
  createAttendance: (...args: unknown[]) => createAttendance(...args),
}));

import {
  createAttendanceWithOfflineFallback,
  cacheAttendanceList,
  getCachedAttendances,
  saveAttendanceOffline,
} from '../attendanceOffline';
import type { CreateAttendanceInput, AttendanceListItem } from '../../types';

const input = {
  person_id: 'person-1',
  service_type_id: 'svc-1',
  professional_id: 'prof-1',
} as unknown as CreateAttendanceInput;

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

describe('createAttendanceWithOfflineFallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    createAttendance.mockReset();
    setOnline(true);
  });

  it('calls the API and returns { offline: false } when online', async () => {
    createAttendance.mockResolvedValue({ id: 'srv-1', person_id: 'person-1' });
    const result = await createAttendanceWithOfflineFallback(input);
    expect(createAttendance).toHaveBeenCalledOnce();
    expect(result.offline).toBe(false);
    expect(result.id).toBe('srv-1');
    expect(await db.syncQueue.count()).toBe(0);
  });

  it('queues to IndexedDB and returns { offline: true } when offline', async () => {
    setOnline(false);
    const result = await createAttendanceWithOfflineFallback(input, 'Maria', 'Consulta');
    expect(createAttendance).not.toHaveBeenCalled();
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
    const cached = await db.attendances.get(result.id);
    expect(cached?.syncStatus).toBe('pending');
    const item = cached?.data as unknown as AttendanceListItem;
    expect(item.person_name).toBe('Maria');
    expect(item.service_type).toBe('Consulta');
    expect(item.status).toBe('SCHEDULED');
  });

  it('queues an entity_type=attendance sync-queue item carrying sync_id', async () => {
    setOnline(false);
    const result = await createAttendanceWithOfflineFallback(input);
    const queued = await db.syncQueue.toArray();
    expect(queued[0]?.entityType).toBe('attendance');
    expect(queued[0]?.action).toBe('create');
    expect((queued[0]?.data as { sync_id: string }).sync_id).toBe(result.id);
  });

  it('falls back to offline queue when the API throws a network error', async () => {
    createAttendance.mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createAttendanceWithOfflineFallback(input);
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
  });

  it('re-throws non-network API errors without queuing', async () => {
    const apiError = Object.assign(new Error('API error: 409'), {
      name: 'ApiError',
      status: 409,
    });
    createAttendance.mockRejectedValue(apiError);
    await expect(createAttendanceWithOfflineFallback(input)).rejects.toBe(apiError);
    expect(await db.syncQueue.count()).toBe(0);
  });
});

describe('cacheAttendanceList / getCachedAttendances', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('round-trips an attendance list through the cache as synced records', async () => {
    const list: AttendanceListItem[] = [
      {
        id: 'a-1',
        person_id: 'p-1',
        person_name: 'João',
        service_type: 'Consulta',
        status: 'SCHEDULED',
        attendance_date: '2026-06-30',
      },
    ];
    await cacheAttendanceList(list);
    const cached = await getCachedAttendances();
    expect(cached).toHaveLength(1);
    expect(cached[0]?.person_name).toBe('João');
    expect((await db.attendances.get('a-1'))?.syncStatus).toBe('synced');
  });
});

describe('saveAttendanceOffline', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('stores a valid AttendanceListItem shape so the list renders offline', async () => {
    const id = await saveAttendanceOffline(input, 'Ana', 'Psicologia');
    const cached = (await db.attendances.get(id))?.data as unknown as AttendanceListItem;
    expect(cached.id).toBe(id);
    expect(cached.person_id).toBe('person-1');
    expect(cached.person_name).toBe('Ana');
    expect(cached.service_type).toBe('Psicologia');
    expect(cached.status).toBe('SCHEDULED');
  });

  it('carries campaign_id into the sync-queue payload when provided', async () => {
    const withCampaign: CreateAttendanceInput = {
      person_id: 'person-1',
      service_type_id: 'svc-1',
      professional_id: 'prof-1',
      campaign_id: 'camp-1',
    };
    await saveAttendanceOffline(withCampaign);
    const queued = await db.syncQueue.toArray();
    expect((queued[0]?.data as { campaign_id?: string }).campaign_id).toBe('camp-1');
  });

  it('omits campaign_id from the sync-queue payload when not provided', async () => {
    await saveAttendanceOffline(input);
    const queued = await db.syncQueue.toArray();
    expect('campaign_id' in (queued[0]?.data ?? {})).toBe(false);
  });
});
