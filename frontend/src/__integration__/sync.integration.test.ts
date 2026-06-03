/**
 * Integration test: client-side sync contract against the Sprint 4 backend.
 *
 * The frontend does not yet have a sync drainer hook or push/pull API client
 * (that ships in the follow-up PR). This file exists to pin the wire-level
 * contract on the consumer side so the future drainer hook lands against a
 * stable, test-backed boundary.
 *
 * Coverage today:
 * - POST /api/v1/sync/push happy path + 413 batch_too_large mapping
 * - GET /api/v1/sync/pull happy path + 400 missing_since mapping
 *
 * When the drainer hook is written, it will import these MSW fixtures and
 * extend the assertions to cover retry/backoff/cursor advancement.
 */

import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';

import { apiClient, ApiError } from '../api/client';
import { API_BASE, server } from './server';
import './setup';

const SYNC_PUSH = '/sync/push';
const SYNC_PULL = '/sync/pull';

interface SyncPushResult {
  sync_id: string;
  status: 'created' | 'conflict' | 'error';
  server_id?: string;
  message?: string;
}

interface SyncPushResponse {
  results: SyncPushResult[];
  server_timestamp: string;
}

interface SyncPullResponse {
  records: Array<{ entity_type: string; id: string; data: Record<string, unknown>; updated_at: string }>;
  server_timestamp: string;
  has_more: boolean;
  next_since?: string;
}

describe('integration: sync push/pull contract', () => {
  it('POST /sync/push returns per-record results', async () => {
    server.use(
      http.post(`${API_BASE}${SYNC_PUSH}`, async ({ request }) => {
        const body = (await request.json()) as { records: Array<{ sync_id: string }> };
        return HttpResponse.json({
          results: body.records.map((r) => ({
            sync_id: r.sync_id,
            status: 'created' as const,
            server_id: 'server-' + r.sync_id,
          })),
          server_timestamp: new Date().toISOString(),
        });
      }),
    );

    const syncID = '11111111-1111-1111-1111-111111111111';
    const resp = await apiClient<SyncPushResponse>(SYNC_PUSH, {
      method: 'POST',
      body: JSON.stringify({
        records: [
          {
            entity_type: 'person',
            sync_id: syncID,
            data: { full_name: 'Maria', document_type: 'CPF', nationality: 'BRA' },
          },
        ],
      }),
    });

    expect(resp.results).toHaveLength(1);
    expect(resp.results[0]?.sync_id).toBe(syncID);
    expect(resp.results[0]?.status).toBe('created');
    expect(resp.results[0]?.server_id).toBe('server-' + syncID);
  });

  it('POST /sync/push maps 413 batch_too_large to ApiError', async () => {
    server.use(
      http.post(`${API_BASE}${SYNC_PUSH}`, () =>
        HttpResponse.json(
          { error: 'batch_too_large', message: 'sync batch exceeds maximum size' },
          { status: 413 },
        ),
      ),
    );

    await expect(
      apiClient(SYNC_PUSH, {
        method: 'POST',
        body: JSON.stringify({ records: [] }),
      }),
    ).rejects.toMatchObject({
      name: 'ApiError',
      status: 413,
      body: { error: 'batch_too_large' },
    });
  });

  it('GET /sync/pull happy path returns records + server_timestamp', async () => {
    server.use(
      http.get(`${API_BASE}${SYNC_PULL}`, ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get('since')).toBe('2026-05-01T00:00:00Z');
        expect(url.searchParams.get('entity_types')).toBe('person,triage');
        return HttpResponse.json({
          records: [
            {
              entity_type: 'person',
              id: '22222222-2222-2222-2222-222222222222',
              data: { full_name: 'Carlos' },
              updated_at: '2026-05-01T12:00:00Z',
            },
          ],
          server_timestamp: '2026-05-01T13:00:00Z',
          has_more: false,
        });
      }),
    );

    const resp = await apiClient<SyncPullResponse>(
      `${SYNC_PULL}?since=2026-05-01T00:00:00Z&entity_types=person,triage`,
    );
    expect(resp.records).toHaveLength(1);
    expect(resp.records[0]?.entity_type).toBe('person');
    expect(resp.has_more).toBe(false);
  });

  it('GET /sync/pull missing since maps to ApiError 400', async () => {
    server.use(
      http.get(`${API_BASE}${SYNC_PULL}`, () =>
        HttpResponse.json(
          { error: 'missing_since', message: "the 'since' query parameter is required" },
          { status: 400 },
        ),
      ),
    );

    let caught: unknown;
    try {
      await apiClient(SYNC_PULL);
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(ApiError);
    expect((caught as ApiError).status).toBe(400);
    expect((caught as ApiError).body).toMatchObject({ error: 'missing_since' });
  });
});
