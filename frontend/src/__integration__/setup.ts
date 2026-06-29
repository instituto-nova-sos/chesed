/**
 * Per-test-run lifecycle for the MSW server. Imported by every integration
 * test file so the harness boots once and resets between tests.
 *
 * Also installs a stub for `useAuthStore.getToken` so the API client doesn't
 * try to talk to a real Keycloak instance during the test.
 */

import { afterAll, afterEach, beforeAll, vi } from 'vitest';
import { server } from './server';
import { useAuthStore } from '../store/authStore';

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
  vi.spyOn(useAuthStore.getState(), 'getToken').mockResolvedValue('test-token');
});

afterEach(() => {
  server.resetHandlers();
});

afterAll(() => {
  server.close();
  vi.restoreAllMocks();
});
