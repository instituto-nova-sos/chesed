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
];

export const server = setupServer(...handlers);
