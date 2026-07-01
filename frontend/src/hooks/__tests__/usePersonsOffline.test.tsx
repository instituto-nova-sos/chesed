import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { renderHook, waitFor } from '@testing-library/react';
import { db } from '../../offline/db';

const listPersons = vi.fn();
vi.mock('../../api/persons', () => ({
  listPersons: (...args: unknown[]) => listPersons(...args),
}));

import { usePersons } from '../usePersons';

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

const serverPage = {
  data: [{ id: 'srv-1', full_name: 'Server Person' }],
  pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
};

describe('usePersons offline fallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    listPersons.mockReset();
    setOnline(true);
  });

  it('caches the server list on a successful online fetch', async () => {
    listPersons.mockResolvedValue(serverPage);
    const { result } = renderHook(() => usePersons());
    await waitFor(() => expect(result.current.persons).toHaveLength(1));
    expect(await db.persons.get('srv-1')).toBeDefined();
  });

  it('shows cached persons (including pending offline records) when offline', async () => {
    // Seed a synced (previously cached) person and a pending offline one.
    await db.persons.bulkPut([
      { id: 'srv-1', data: { id: 'srv-1', full_name: 'Cached' }, syncStatus: 'synced', localCreatedAt: 'x' },
      { id: 'off-1', data: { id: 'off-1', full_name: 'My Offline Person' }, syncStatus: 'pending', localCreatedAt: 'y' },
    ]);
    setOnline(false);
    listPersons.mockRejectedValue(new TypeError('Failed to fetch'));

    const { result } = renderHook(() => usePersons());

    await waitFor(() => expect(result.current.persons.length).toBe(2));
    const names = result.current.persons.map((p) => p.full_name);
    expect(names).toContain('My Offline Person');
    expect(result.current.error).toBeNull();
  });
});
