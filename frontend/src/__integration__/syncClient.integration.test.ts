/**
 * Integration test for the sync API client (api/sync.ts) against MSW.
 *
 * Exercises the real apiClient → syncPush/syncPull chain at the fetch boundary,
 * asserting the request shape the backend expects (query string, method, body)
 * and the error mapping to ApiError. Complements syncEngine's IndexedDB unit
 * tests, which cover the drain/merge logic.
 */

import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';

import { syncPush, syncPull } from '../api/sync';
import { ApiError } from '../api/client';
import { API_BASE, server } from './server';
import './setup';

describe('integration: sync API client', () => {
  it('syncPush POSTs { records } and returns per-record results', async () => {
    let capturedBody: unknown;
    server.use(
      http.post(`${API_BASE}/sync/push`, async ({ request }) => {
        capturedBody = await request.json();
        expect(request.headers.get('Authorization')).toMatch(/^Bearer /);
        return HttpResponse.json({
          results: [{ sync_id: 's1', status: 'created', server_id: 'srv-s1' }],
          server_timestamp: '2026-01-01T00:00:00Z',
        });
      }),
    );

    const resp = await syncPush([
      { entity_type: 'person', sync_id: 's1', data: { full_name: 'Maria' } },
    ]);

    expect(capturedBody).toEqual({
      records: [{ entity_type: 'person', sync_id: 's1', data: { full_name: 'Maria' } }],
    });
    expect(resp.results[0]?.status).toBe('created');
    expect(resp.results[0]?.server_id).toBe('srv-s1');
  });

  it('syncPush maps a 413 batch_too_large to ApiError', async () => {
    server.use(
      http.post(`${API_BASE}/sync/push`, () =>
        HttpResponse.json({ error: 'batch_too_large' }, { status: 413 }),
      ),
    );

    await expect(syncPush([])).rejects.toMatchObject({
      name: 'ApiError',
      status: 413,
    });
  });

  it('syncPull sends the since cursor and returns records', async () => {
    let capturedURL = '';
    server.use(
      http.get(`${API_BASE}/sync/pull`, ({ request }) => {
        capturedURL = request.url;
        return HttpResponse.json({
          records: [
            { entity_type: 'person', id: 'p1', data: { id: 'p1' }, updated_at: '2026-01-02T00:00:00Z' },
          ],
          server_timestamp: '2026-01-02T00:00:00Z',
          has_more: false,
        });
      }),
    );

    const resp = await syncPull('2026-01-01T00:00:00Z', ['person'], 100);

    const url = new URL(capturedURL);
    expect(url.searchParams.get('since')).toBe('2026-01-01T00:00:00Z');
    expect(url.searchParams.get('entity_types')).toBe('person');
    expect(url.searchParams.get('limit')).toBe('100');
    expect(resp.records).toHaveLength(1);
    expect(resp.has_more).toBe(false);
  });

  it('syncPull maps a 400 missing_since to ApiError', async () => {
    server.use(
      http.get(`${API_BASE}/sync/pull`, () =>
        HttpResponse.json({ error: 'missing_since' }, { status: 400 }),
      ),
    );

    await expect(syncPull('')).rejects.toBeInstanceOf(ApiError);
  });
});
