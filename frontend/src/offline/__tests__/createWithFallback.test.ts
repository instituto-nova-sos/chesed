import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { db } from '../db';

const createPerson = vi.fn();
vi.mock('../../api/persons', () => ({
  createPerson: (...args: unknown[]) => createPerson(...args),
}));

import { createPersonWithOfflineFallback } from '../personOffline';
import type { CreatePersonInput } from '../../types';

const input = {
  full_name: 'Maria',
  document_type: 'CPF',
} as unknown as CreatePersonInput;

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

describe('createPersonWithOfflineFallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    createPerson.mockReset();
    setOnline(true);
  });

  it('calls the API and returns { offline: false } when online', async () => {
    createPerson.mockResolvedValue({ id: 'srv-1', full_name: 'Maria' });
    const result = await createPersonWithOfflineFallback(input);
    expect(createPerson).toHaveBeenCalledOnce();
    expect(result.offline).toBe(false);
    expect(result.id).toBe('srv-1');
    expect(await db.syncQueue.count()).toBe(0);
  });

  it('queues to IndexedDB and returns { offline: true } when offline', async () => {
    setOnline(false);
    const result = await createPersonWithOfflineFallback(input);
    expect(createPerson).not.toHaveBeenCalled();
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
    expect((await db.persons.get(result.id))?.syncStatus).toBe('pending');
  });

  it('falls back to offline queue when the API throws a network error', async () => {
    createPerson.mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createPersonWithOfflineFallback(input);
    expect(result.offline).toBe(true);
    expect(await db.syncQueue.count()).toBe(1);
  });

  it('re-throws non-network API errors (validation, conflict) without queuing', async () => {
    const apiError = Object.assign(new Error('API error: 409'), { name: 'ApiError', status: 409 });
    createPerson.mockRejectedValue(apiError);
    await expect(createPersonWithOfflineFallback(input)).rejects.toBe(apiError);
    expect(await db.syncQueue.count()).toBe(0);
  });
});
