import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';

const createTriage = vi.fn();
vi.mock('../../api/triages', () => ({
  createTriage: (...args: unknown[]) => createTriage(...args),
}));

import {
  createTriageWithOfflineFallback,
  cacheTriageList,
  getCachedTriages,
  saveTriageOffline,
} from '../triageOffline';
import type { CreateTriageInput, TriageListItem } from '../../types';

const input = {
  person_id: 'person-1',
  main_complaint: 'Dor de cabeça',
  requested_service_types: ['svc-1'],
} as unknown as CreateTriageInput;

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

describe('createTriageWithOfflineFallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    createTriage.mockReset();
    setOnline(true);
  });

  it('calls the API and returns { offline: false } when online', async () => {
    createTriage.mockResolvedValue({ id: 'srv-1', person_id: 'person-1' });
    const result = await createTriageWithOfflineFallback(input);
    expect(createTriage).toHaveBeenCalledOnce();
    expect(result.offline).toBe(false);
    expect(result.id).toBe('srv-1');
    expect(await db.syncQueue.count()).toBe(0);
  });

  it('queues to IndexedDB and returns { offline: true } when offline', async () => {
    setOnline(false);
    const result = await createTriageWithOfflineFallback(input, 'Maria');
    expect(createTriage).not.toHaveBeenCalled();
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
    const cached = await db.triages.get(result.id);
    expect(cached?.syncStatus).toBe('pending');
    expect((cached?.data as unknown as TriageListItem).person_name).toBe('Maria');
  });

  it('queues an entity_type=triage sync-queue item carrying sync_id', async () => {
    setOnline(false);
    const result = await createTriageWithOfflineFallback(input);
    const queued = await db.syncQueue.toArray();
    expect(queued[0]?.entityType).toBe('triage');
    expect(queued[0]?.action).toBe('create');
    expect((queued[0]?.data as { sync_id: string }).sync_id).toBe(result.id);
  });

  it('falls back to offline queue when the API throws a network error', async () => {
    createTriage.mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createTriageWithOfflineFallback(input);
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
  });

  it('re-throws non-network API errors without queuing', async () => {
    const apiError = Object.assign(new Error('API error: 409'), {
      name: 'ApiError',
      status: 409,
    });
    createTriage.mockRejectedValue(apiError);
    await expect(createTriageWithOfflineFallback(input)).rejects.toBe(apiError);
    expect(await db.syncQueue.count()).toBe(0);
  });
});

describe('cacheTriageList / getCachedTriages', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('round-trips a triage list through the cache as synced records', async () => {
    const list: TriageListItem[] = [
      {
        id: 't-1',
        person_id: 'p-1',
        person_name: 'João',
        main_complaint: 'Febre',
        triage_date: '2026-06-30',
        requested_service_count: 2,
      },
    ];
    await cacheTriageList(list);
    const cached = await getCachedTriages();
    expect(cached).toHaveLength(1);
    expect(cached[0]?.person_name).toBe('João');
    expect((await db.triages.get('t-1'))?.syncStatus).toBe('synced');
  });
});

describe('saveTriageOffline', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('stores a valid TriageListItem shape so the list renders offline', async () => {
    const id = await saveTriageOffline(input, 'Ana');
    const cached = (await db.triages.get(id))?.data as unknown as TriageListItem;
    expect(cached.id).toBe(id);
    expect(cached.person_id).toBe('person-1');
    expect(cached.person_name).toBe('Ana');
    expect(cached.main_complaint).toBe('Dor de cabeça');
    expect(cached.requested_service_count).toBe(1);
  });

  it('defaults requested_service_count to 0 when no services are requested', async () => {
    const noServices = {
      person_id: 'person-1',
      main_complaint: 'Dor',
    } as unknown as CreateTriageInput;
    const id = await saveTriageOffline(noServices);
    const cached = (await db.triages.get(id))?.data as unknown as TriageListItem;
    expect(cached.requested_service_count).toBe(0);
    expect(cached.person_name).toBe('');
  });
});
