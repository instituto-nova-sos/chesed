import { describe, expect, it, beforeEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import { renderHook, waitFor } from '@testing-library/react';
import { db } from '../../offline/db';

const listAttendances = vi.fn();
vi.mock('../../api/attendances', () => ({
  listAttendances: (...args: unknown[]) => listAttendances(...args),
}));

import { useAttendances } from '../useAttendances';

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { value, configurable: true });
}

const serverPage = {
  data: [
    {
      id: 'srv-1',
      person_id: 'p-1',
      person_name: 'Server Person',
      service_type: 'Consulta',
      status: 'SCHEDULED',
      attendance_date: '2026-06-30',
    },
  ],
  pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
};

describe('useAttendances offline fallback', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    listAttendances.mockReset();
    setOnline(true);
  });

  it('caches the server list on a successful online fetch', async () => {
    listAttendances.mockResolvedValue(serverPage);
    const { result } = renderHook(() => useAttendances());
    await waitFor(() => expect(result.current.attendances).toHaveLength(1));
    expect(await db.attendances.get('srv-1')).toBeDefined();
  });

  it('shows cached attendances (including pending offline records) when offline', async () => {
    await db.attendances.bulkPut([
      {
        id: 'srv-1',
        data: { id: 'srv-1', person_name: 'Cached', service_type: 'x', status: 'SCHEDULED' },
        syncStatus: 'synced',
        localCreatedAt: 'x',
      },
      {
        id: 'off-1',
        data: { id: 'off-1', person_name: 'My Offline Attendance', service_type: 'y', status: 'SCHEDULED' },
        syncStatus: 'pending',
        localCreatedAt: 'y',
      },
    ]);
    setOnline(false);
    listAttendances.mockRejectedValue(new TypeError('Failed to fetch'));

    const { result } = renderHook(() => useAttendances());

    await waitFor(() => expect(result.current.attendances.length).toBe(2));
    const names = result.current.attendances.map((a) => a.person_name);
    expect(names).toContain('My Offline Attendance');
    expect(result.current.error).toBeNull();
  });
});
