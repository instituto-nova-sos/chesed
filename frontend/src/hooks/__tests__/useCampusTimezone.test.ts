import { beforeEach, describe, expect, it, vi } from 'vitest';

const keycloakMock = vi.hoisted(() => ({
  token: undefined as string | undefined,
  tokenParsed: undefined as Record<string, unknown> | undefined,
  updateToken: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  init: vi.fn(),
  onTokenExpired: undefined as (() => void) | undefined,
}));

vi.mock('../../auth/keycloak', () => ({ keycloak: keycloakMock }));

import { renderHook } from '@testing-library/react';
import { useAuthStore } from '../../store/authStore';
import { useCampusTimezone } from '../useCampusTimezone';

const INITIAL_STATE = useAuthStore.getState();

describe('useCampusTimezone', () => {
  beforeEach(() => {
    useAuthStore.setState(INITIAL_STATE, true);
  });

  it('returns undefined when no campus timezone is available in context', () => {
    const { result } = renderHook(() => useCampusTimezone());
    expect(result.current).toBeUndefined();
  });

  it('returns the campus timezone when the auth store exposes one', () => {
    useAuthStore.setState({
      campusTimezone: 'America/Sao_Paulo',
    } as Partial<ReturnType<typeof useAuthStore.getState>>);

    const { result } = renderHook(() => useCampusTimezone());
    expect(result.current).toBe('America/Sao_Paulo');
  });
});
