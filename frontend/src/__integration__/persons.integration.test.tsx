/**
 * Integration test: usePersons hook against an MSW-backed API surface.
 *
 * What this exercises that a unit test does not:
 * - The real `apiClient` HTTP path (fetch + auth token header + JSON parsing
 *   + ApiError mapping).
 * - URL serialization for query params (`q`, `page`, `agreement_status`).
 * - The debounce + state-update flow in `usePersons`.
 *
 * MSW intercepts at the network layer so any drift between the API client
 * and the documented contract trips the test, in the same way the backend
 * integration tests catch SQL contract drift.
 */

import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { usePersons } from '../hooks/usePersons';
import { API_BASE, server } from './server';
import './setup';

describe('integration: usePersons + apiClient + MSW', () => {
  it('loads the default page on mount', async () => {
    const { result } = renderHook(() => usePersons());

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.persons).toHaveLength(1);
    expect(result.current.persons[0]?.full_name).toBe('Default Person');
    expect(result.current.pagination.total).toBe(1);
  });

  it('forwards the search query to the API (debounced)', async () => {
    const { result } = renderHook(() => usePersons());
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.search('Maria'));

    // The debounce fires ~300ms later, then a second fetch resolves.
    // Wait for the actual data to update instead of the loading flag,
    // which racy-flips between true/false during the transition.
    await waitFor(
      () => expect(result.current.persons[0]?.full_name).toBe('Match for "Maria"'),
      { timeout: 2000 },
    );
  });

  it('surfaces a server error through the hook state', async () => {
    server.use(
      http.get(`${API_BASE}/persons`, () =>
        HttpResponse.json({ error: 'internal', message: 'boom' }, { status: 500 }),
      ),
    );

    const { result } = renderHook(() => usePersons());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.error).not.toBeNull();
    expect(result.current.persons).toEqual([]);
  });

  it('includes the Bearer token on every request', async () => {
    let receivedAuth: string | null = null;
    server.use(
      http.get(`${API_BASE}/persons`, ({ request }) => {
        receivedAuth = request.headers.get('authorization');
        return HttpResponse.json({
          data: [],
          pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
        });
      }),
    );

    const { result } = renderHook(() => usePersons());
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(receivedAuth).toBe('Bearer test-token');
  });
});
