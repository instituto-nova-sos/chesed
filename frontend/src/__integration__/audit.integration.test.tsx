/**
 * Integration test: audit log viewer API surface (S11.6) against MSW.
 *
 * Exercises the real `apiClient` path for the audit contract: the query-string
 * serialization of the filter set and ApiError mapping for a 500 response.
 * Handlers are registered per-test via `server.use(...)` so these tests do not
 * depend on the shared server wiring order.
 */

import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { listAuditLogs } from '../api/audit';
import { API_BASE, server } from './server';
import './setup';

const PAGE = {
  data: [
    {
      id: '00000000-0000-0000-0000-0000000000a1',
      user_email: 'maria@example.com',
      action_type: 'UPDATE',
      entity_type: 'person',
      entity_id: '00000000-0000-0000-0000-0000000000b1',
      description: 'Updated phone number',
      old_values: { phone: '+55 11 88888-0000' },
      new_values: { phone: '+55 11 99999-0000' },
      ip_address: '192.168.1.1',
      timestamp: '2026-04-02T10:30:00Z',
    },
  ],
  pagination: { page: 1, per_page: 50, total: 1, total_pages: 1 },
};

describe('integration: audit logs + apiClient + MSW', () => {
  it('loads a page of audit entries', async () => {
    server.use(
      http.get(`${API_BASE}/audit/logs`, () => HttpResponse.json(PAGE)),
    );

    const page = await listAuditLogs({ start: '2026-01-01', end: '2026-12-31' });

    expect(page.data).toHaveLength(1);
    expect(page.data[0]?.user_email).toBe('maria@example.com');
    expect(page.pagination.total).toBe(1);
  });

  it('serializes every filter as a query parameter', async () => {
    let params: URLSearchParams | null = null;
    server.use(
      http.get(`${API_BASE}/audit/logs`, ({ request }) => {
        params = new URL(request.url).searchParams;
        return HttpResponse.json(PAGE);
      }),
    );

    await listAuditLogs({
      start: '2026-01-01',
      end: '2026-03-31',
      user_id: 'u-1',
      entity_type: 'person',
      action_type: 'UPDATE',
      page: 2,
      per_page: 25,
    });

    const seen = params as unknown as URLSearchParams;
    expect(seen.get('start')).toBe('2026-01-01');
    expect(seen.get('end')).toBe('2026-03-31');
    expect(seen.get('user_id')).toBe('u-1');
    expect(seen.get('entity_type')).toBe('person');
    expect(seen.get('action_type')).toBe('UPDATE');
    expect(seen.get('page')).toBe('2');
    expect(seen.get('per_page')).toBe('25');
  });

  it('omits unset optional filters from the query string', async () => {
    let params: URLSearchParams | null = null;
    server.use(
      http.get(`${API_BASE}/audit/logs`, ({ request }) => {
        params = new URL(request.url).searchParams;
        return HttpResponse.json(PAGE);
      }),
    );

    await listAuditLogs({ start: '2026-01-01', end: '2026-12-31' });

    const seen = params as unknown as URLSearchParams;
    expect(seen.get('user_id')).toBeNull();
    expect(seen.get('entity_type')).toBeNull();
    expect(seen.get('action_type')).toBeNull();
  });

  it('maps a 500 response to an ApiError with status', async () => {
    server.use(
      http.get(`${API_BASE}/audit/logs`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    await expect(
      listAuditLogs({ start: '2026-01-01', end: '2026-12-31' }),
    ).rejects.toMatchObject({ name: 'ApiError', status: 500 });
  });
});
