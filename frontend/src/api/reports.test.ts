import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  downloadAttendancesCSV,
  getAttendanceReport,
  suggestedCSVFilename,
} from './reports';

vi.mock('../store/authStore', () => ({
  useAuthStore: {
    getState: () => ({
      getToken: () => Promise.resolve('test-token'),
    }),
  },
}));

const ORIGINAL_FETCH = globalThis.fetch;

describe('reports api', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });

  afterEach(() => {
    globalThis.fetch = ORIGINAL_FETCH;
    vi.restoreAllMocks();
  });

  describe('getAttendanceReport', () => {
    it('sends start/end query params and bearer token', async () => {
      const payload = {
        period: { start: '2026-01-01', end: '2026-01-31' },
        total_attendances: 5,
        unique_persons: 3,
        by_status: { COMPLETED: 5 },
        by_service_type: [],
        by_month: [],
      };
      const fetchMock = vi.mocked(globalThis.fetch);
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(payload),
      } as Response);

      const result = await getAttendanceReport({
        start: '2026-01-01',
        end: '2026-01-31',
      });

      expect(result).toEqual(payload);
      expect(fetchMock).toHaveBeenCalledOnce();
      const [url, init] = fetchMock.mock.calls[0]!;
      expect(url).toContain('/reports/attendances?');
      expect(url).toContain('start=2026-01-01');
      expect(url).toContain('end=2026-01-31');
      const headers = (init?.headers ?? {}) as Record<string, string>;
      expect(headers.Authorization).toBe('Bearer test-token');
    });
  });

  describe('downloadAttendancesCSV', () => {
    it('returns blob from successful response with format=csv', async () => {
      const blob = new Blob(['id,name\n'], { type: 'text/csv' });
      const fetchMock = vi.mocked(globalThis.fetch);
      fetchMock.mockResolvedValueOnce({
        ok: true,
        blob: () => Promise.resolve(blob),
      } as Response);

      const result = await downloadAttendancesCSV({
        start: '2026-01-01',
        end: '2026-01-31',
      });

      expect(result).toBe(blob);
      const [url, init] = fetchMock.mock.calls[0]!;
      expect(url).toContain('/reports/attendances/export?');
      expect(url).toContain('format=csv');
      const headers = (init?.headers ?? {}) as Record<string, string>;
      expect(headers.Authorization).toBe('Bearer test-token');
    });

    it('throws ApiError on non-ok response', async () => {
      const fetchMock = vi.mocked(globalThis.fetch);
      fetchMock.mockResolvedValueOnce({
        ok: false,
        status: 403,
        json: () => Promise.resolve({ error: 'forbidden' }),
      } as Response);

      await expect(
        downloadAttendancesCSV({ start: '2026-01-01', end: '2026-01-31' }),
      ).rejects.toMatchObject({ status: 403 });
    });
  });

  describe('suggestedCSVFilename', () => {
    it('builds a filename from the period', () => {
      expect(
        suggestedCSVFilename({ start: '2026-01-01', end: '2026-03-31' }),
      ).toBe('attendances_2026-01-01_2026-03-31.csv');
    });
  });
});
