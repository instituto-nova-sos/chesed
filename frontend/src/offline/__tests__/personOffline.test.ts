import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { CreatePersonInput, PersonListItem } from '../../types';
import type { LocalPerson, SyncQueueItem } from '../db';

const { personsTable, syncQueueTable } = vi.hoisted(() => ({
  personsTable: {
    bulkPut: vi.fn(),
    put: vi.fn(),
    toArray: vi.fn(),
    get: vi.fn(),
  },
  syncQueueTable: {
    add: vi.fn(),
    count: vi.fn(),
  },
}));

vi.mock('../db', () => ({
  db: {
    persons: personsTable,
    syncQueue: syncQueueTable,
  },
}));

import {
  cachePersonList,
  getCachedPersons,
  getOfflinePerson,
  getPendingSyncCount,
  savePersonOffline,
} from '../personOffline';

const ISO_8601 =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;

function buildPersonInput(
  overrides: Partial<CreatePersonInput> = {},
): CreatePersonInput {
  return {
    full_name: 'Maria Silva',
    document_type: 'CPF',
    nationality: 'BRA',
    ...overrides,
  } as CreatePersonInput;
}

describe('personOffline', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('cachePersonList', () => {
    it('maps server persons to synced LocalPerson rows and bulk-puts them', async () => {
      const persons = [
        { id: 'p1', full_name: 'Ana' },
        { id: 'p2', full_name: 'Bruno' },
      ] as unknown as PersonListItem[];

      await cachePersonList(persons);

      expect(personsTable.bulkPut).toHaveBeenCalledOnce();
      const stored = personsTable.bulkPut.mock.calls[0]![0] as LocalPerson[];
      expect(stored).toHaveLength(2);
      for (const [index, row] of stored.entries()) {
        expect(row.id).toBe(persons[index]!.id);
        expect(row.syncStatus).toBe('synced');
        expect(row.data).toBe(persons[index]);
        expect(row.localCreatedAt).toMatch(ISO_8601);
      }
    });

    it('writes an empty array when there are no persons', async () => {
      await cachePersonList([]);
      expect(personsTable.bulkPut).toHaveBeenCalledWith([]);
    });
  });

  describe('getCachedPersons', () => {
    it('unwraps the stored data field from every row', async () => {
      const rows: LocalPerson[] = [
        {
          id: 'p1',
          data: { id: 'p1', full_name: 'Ana' },
          syncStatus: 'synced',
          localCreatedAt: '2026-01-01T00:00:00.000Z',
        },
      ];
      personsTable.toArray.mockResolvedValueOnce(rows);

      const result = await getCachedPersons();

      expect(result).toEqual([{ id: 'p1', full_name: 'Ana' }]);
    });
  });

  describe('savePersonOffline', () => {
    it('stores a pending person carrying its own id', async () => {
      const input = buildPersonInput();

      const id = await savePersonOffline(input);

      expect(typeof id).toBe('string');
      expect(personsTable.put).toHaveBeenCalledOnce();
      const stored = personsTable.put.mock.calls[0]![0] as LocalPerson;
      expect(stored.id).toBe(id);
      expect(stored.syncStatus).toBe('pending');
      expect(stored.localCreatedAt).toMatch(ISO_8601);
      expect((stored.data as Record<string, unknown>).id).toBe(id);
      expect((stored.data as Record<string, unknown>).full_name).toBe(
        input.full_name,
      );
      // The cached data must be a valid PersonListItem so the list/card render
      // offline without crashing on missing fields (e.g. roles.length).
      expect((stored.data as Record<string, unknown>).roles).toEqual([]);
      expect((stored.data as Record<string, unknown>).is_active).toBe(true);
    });

    it('enqueues a create sync item whose sync_id matches the person id', async () => {
      const input = buildPersonInput({ full_name: 'Carlos' });

      const id = await savePersonOffline(input);

      expect(syncQueueTable.add).toHaveBeenCalledOnce();
      const queued = syncQueueTable.add.mock.calls[0]![0] as SyncQueueItem;
      expect(queued.entityType).toBe('person');
      expect(queued.entityId).toBe(id);
      expect(queued.action).toBe('create');
      expect(queued.retryCount).toBe(0);
      expect(queued.lastError).toBeUndefined();
      expect(queued.createdAt).toMatch(ISO_8601);
      expect((queued.data as Record<string, unknown>).sync_id).toBe(id);
      expect((queued.data as Record<string, unknown>).full_name).toBe('Carlos');
    });

    it('generates a unique id per call', async () => {
      const first = await savePersonOffline(buildPersonInput());
      const second = await savePersonOffline(buildPersonInput());
      expect(first).not.toBe(second);
    });
  });

  describe('getPendingSyncCount', () => {
    it('returns the syncQueue row count', async () => {
      syncQueueTable.count.mockResolvedValueOnce(3);
      await expect(getPendingSyncCount()).resolves.toBe(3);
      expect(syncQueueTable.count).toHaveBeenCalledOnce();
    });
  });

  describe('getOfflinePerson', () => {
    it('delegates to persons.get with the given id', async () => {
      const row: LocalPerson = {
        id: 'p9',
        data: { id: 'p9' },
        syncStatus: 'pending',
        localCreatedAt: '2026-01-01T00:00:00.000Z',
      };
      personsTable.get.mockResolvedValueOnce(row);

      const result = await getOfflinePerson('p9');

      expect(personsTable.get).toHaveBeenCalledWith('p9');
      expect(result).toBe(row);
    });

    it('returns undefined when the person is not cached', async () => {
      personsTable.get.mockResolvedValueOnce(undefined);
      await expect(getOfflinePerson('missing')).resolves.toBeUndefined();
    });
  });
});
