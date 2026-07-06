/**
 * Integration test: reports API surface (E10 dashboards + attendance report)
 * against MSW.
 *
 * Exercises the real `apiClient` / hook path for the E10 contract: the
 * dashboard metrics fetch feeding `useDashboard`, the attendance report filter
 * query-string serialization, and ApiError mapping for a 500 dashboard
 * response. Handlers are registered per-test via `server.use(...)` so these
 * tests do not depend on the shared server wiring order (setup uses
 * onUnhandledRequest: 'error').
 */

import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { getAttendanceReport } from '../api/reports';
import { useDashboard } from '../hooks/useDashboard';
import { API_BASE, server } from './server';
import './setup';

const DASHBOARD = {
  total_persons: 42,
  attendances_this_month: 8,
  upcoming_scheduled: 3,
  active_campaigns: 2,
  attendances_by_status: {
    COMPLETED: 30,
    SCHEDULED: 5,
    IN_PROGRESS: 4,
    CANCELLED: 1,
  },
  recent_months: [
    { month: '2026-06', count: 9 },
    { month: '2026-07', count: 8 },
  ],
};

describe('integration: dashboard metrics + useDashboard + MSW', () => {
  it('loads dashboard metrics on mount', async () => {
    server.use(
      http.get(`${API_BASE}/reports/dashboard`, () =>
        HttpResponse.json(DASHBOARD),
      ),
    );

    const { result } = renderHook(() => useDashboard());

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.data?.total_persons).toBe(42);
    expect(result.current.data?.active_campaigns).toBe(2);
    expect(result.current.data?.recent_months).toHaveLength(2);
  });

  it('surfaces a server error through the hook state', async () => {
    server.use(
      http.get(`${API_BASE}/reports/dashboard`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    const { result } = renderHook(() => useDashboard());

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).not.toBeNull();
    expect(result.current.data).toBeNull();
  });
});

describe('integration: attendance report + apiClient + MSW', () => {
  it('carries the service_type_id filter as a query parameter', async () => {
    let seenServiceType: string | null = null;
    let seenProfessional: string | null = null;
    server.use(
      http.get(`${API_BASE}/reports/attendances`, ({ request }) => {
        const params = new URL(request.url).searchParams;
        seenServiceType = params.get('service_type_id');
        seenProfessional = params.get('professional_id');
        return HttpResponse.json({
          period: { start: '2026-07-01', end: '2026-07-31' },
          total_attendances: 12,
          unique_persons: 10,
          by_status: { COMPLETED: 12 },
          by_service_type: [{ service_type: 'LEGAL', count: 12 }],
          by_month: [{ month: '2026-07', count: 12 }],
          by_professional: [],
        });
      }),
    );

    const report = await getAttendanceReport({
      start: '2026-07-01',
      end: '2026-07-31',
      service_type_id: 'st-1',
    });

    expect(report.total_attendances).toBe(12);
    expect(seenServiceType).toBe('st-1');
    expect(seenProfessional).toBeNull();
  });

  it('maps a 500 report response to an ApiError with status', async () => {
    server.use(
      http.get(`${API_BASE}/reports/attendances`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    await expect(
      getAttendanceReport({ start: '2026-07-01', end: '2026-07-31' }),
    ).rejects.toMatchObject({ name: 'ApiError', status: 500 });
  });
});
