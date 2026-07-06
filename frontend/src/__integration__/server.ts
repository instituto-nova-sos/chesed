/**
 * Mock Service Worker setup for frontend integration tests.
 *
 * Integration tests intercept real `fetch` calls at the network layer via
 * MSW. Unlike unit tests that mock individual modules, integration tests
 * exercise the full API client → hook → component chain against a realistic
 * HTTP boundary.
 *
 * Why MSW (not a Vite proxy or a live backend):
 * - Tests stay hermetic — no Docker or stack-up dance per test run.
 * - The intercept boundary is HTTP, so we catch contract drift in the API
 *   client (request shape, headers, query string serialization) — the
 *   same kinds of bugs a live backend would catch.
 * - Mocked responses live next to the test so the contract is co-located
 *   with the assertion.
 *
 * Default handlers below cover the read paths most other tests reuse;
 * individual tests `.use(...)` their own handlers for the specific case
 * they exercise (errors, edge values, etc.).
 */

import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

export const API_BASE = 'http://localhost:8080/api/v1';

export const handlers = [
  http.get(`${API_BASE}/campaigns`, () =>
    HttpResponse.json({
      data: [
        {
          id: '00000000-0000-0000-0000-00000000c001',
          name: 'Default Campaign',
          campaign_type: 'SOCIAL_ACTION',
          status: 'ACTIVE',
          start_date: '2026-07-10T00:00:00Z',
          location_name: 'Community Center',
        },
      ],
      pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
    }),
  ),
  http.get(`${API_BASE}/donations`, () =>
    HttpResponse.json({
      data: [
        {
          id: '00000000-0000-0000-0000-00000000d001',
          donation_type: 'FINANCIAL',
          amount: 150.0,
          currency: 'BRL',
          donation_date: '2026-07-05',
          donor_name: 'Default Donor',
          campaign_name: null,
        },
      ],
      pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
    }),
  ),
  http.get(`${API_BASE}/persons`, ({ request }) => {
    const url = new URL(request.url);
    const q = url.searchParams.get('q') ?? '';
    const samplePerson = {
      id: '00000000-0000-0000-0000-000000000001',
      full_name: q ? `Match for "${q}"` : 'Default Person',
      document_number: '11111111111',
      phone: null,
      roles: ['VOLUNTEER'],
      is_active: true,
    };
    return HttpResponse.json({
      data: [samplePerson],
      pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
    });
  }),
  http.get(`${API_BASE}/reports/dashboard`, () =>
    HttpResponse.json({
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
        { month: '2026-02', count: 5 },
        { month: '2026-03', count: 7 },
        { month: '2026-04', count: 6 },
        { month: '2026-05', count: 8 },
        { month: '2026-06', count: 9 },
        { month: '2026-07', count: 8 },
      ],
    }),
  ),
  http.get(`${API_BASE}/reports/attendances`, () =>
    HttpResponse.json({
      period: { start: '2026-07-01', end: '2026-07-31' },
      total_attendances: 40,
      unique_persons: 33,
      by_status: { COMPLETED: 30, SCHEDULED: 5, IN_PROGRESS: 4, CANCELLED: 1 },
      by_service_type: [
        { service_type: 'LEGAL', count: 12 },
        { service_type: 'MEDICAL', count: 28 },
      ],
      by_month: [{ month: '2026-07', count: 40 }],
      by_professional: [
        {
          professional_id: '00000000-0000-0000-0000-0000000000a1',
          professional_name: 'Maria Silva',
          count: 25,
        },
        {
          professional_id: '00000000-0000-0000-0000-0000000000a2',
          professional_name: 'João Souza',
          count: 15,
        },
      ],
    }),
  ),
];

export const server = setupServer(...handlers);
