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

vi.mock('../../auth/keycloak', () => ({
  keycloak: keycloakMock,
}));

import { useAuthStore } from '../../store/authStore';

const INITIAL_STATE = useAuthStore.getState();

describe('authStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    keycloakMock.token = undefined;
    keycloakMock.tokenParsed = undefined;
    useAuthStore.setState(INITIAL_STATE, true);
  });

  describe('getToken', () => {
    it('refreshes, stores and returns the current keycloak token', async () => {
      keycloakMock.updateToken.mockResolvedValueOnce(true);
      keycloakMock.token = 'fresh-token';

      const token = await useAuthStore.getState().getToken();

      expect(keycloakMock.updateToken).toHaveBeenCalledWith(30);
      expect(token).toBe('fresh-token');
      expect(useAuthStore.getState().token).toBe('fresh-token');
    });

    it('returns null and stores null when keycloak holds no token', async () => {
      keycloakMock.updateToken.mockResolvedValueOnce(false);
      keycloakMock.token = undefined;

      const token = await useAuthStore.getState().getToken();

      expect(token).toBeNull();
      expect(useAuthStore.getState().token).toBeNull();
    });

    it('logs the user out and triggers login when refresh fails', async () => {
      keycloakMock.updateToken.mockRejectedValueOnce(new Error('expired'));
      useAuthStore.setState({ isAuthenticated: true });

      const token = await useAuthStore.getState().getToken();

      expect(token).toBeNull();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(keycloakMock.login).toHaveBeenCalledOnce();
    });
  });

  describe('setCampusId', () => {
    it('stores the selected campus id', () => {
      useAuthStore.getState().setCampusId('campus-123');
      expect(useAuthStore.getState().campusId).toBe('campus-123');
    });
  });

  describe('logout', () => {
    it('delegates to keycloak.logout with the origin redirect', () => {
      useAuthStore.getState().logout();
      expect(keycloakMock.logout).toHaveBeenCalledWith({
        redirectUri: window.location.origin,
      });
    });
  });

  describe('initialize', () => {
    it('hydrates identity claims from the parsed token when authenticated', async () => {
      keycloakMock.init.mockResolvedValueOnce(true);
      keycloakMock.token = 'session-token';
      keycloakMock.tokenParsed = {
        email: 'user@example.com',
        email_verified: true,
        realm_access: { roles: ['ADMIN'] },
        person_id: 'person-7',
      };

      await useAuthStore.getState().initialize();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.isLoading).toBe(false);
      expect(state.initialized).toBe(true);
      expect(state.email).toBe('user@example.com');
      expect(state.emailVerified).toBe(true);
      expect(state.roles).toEqual(['ADMIN']);
      expect(state.personId).toBe('person-7');
      expect(state.token).toBe('session-token');
    });

    it('registers an onTokenExpired handler that stores the refreshed token', async () => {
      keycloakMock.init.mockResolvedValueOnce(true);
      keycloakMock.tokenParsed = { email: 'user@example.com' };
      keycloakMock.token = 'first-token';

      await useAuthStore.getState().initialize();
      expect(keycloakMock.onTokenExpired).toBeTypeOf('function');

      keycloakMock.updateToken.mockResolvedValueOnce(true);
      keycloakMock.token = 'refreshed-token';
      await keycloakMock.onTokenExpired?.();

      expect(keycloakMock.updateToken).toHaveBeenCalledWith(30);
      expect(useAuthStore.getState().token).toBe('refreshed-token');
    });

    it('onTokenExpired keeps the token when keycloak reports no refresh', async () => {
      keycloakMock.init.mockResolvedValueOnce(true);
      keycloakMock.tokenParsed = { email: 'user@example.com' };
      keycloakMock.token = 'first-token';

      await useAuthStore.getState().initialize();

      keycloakMock.updateToken.mockResolvedValueOnce(false);
      await keycloakMock.onTokenExpired?.();

      expect(useAuthStore.getState().token).toBe('first-token');
    });

    it('onTokenExpired logs out and re-authenticates when the refresh fails', async () => {
      keycloakMock.init.mockResolvedValueOnce(true);
      keycloakMock.tokenParsed = { email: 'user@example.com' };
      keycloakMock.token = 'first-token';

      await useAuthStore.getState().initialize();
      expect(useAuthStore.getState().isAuthenticated).toBe(true);

      keycloakMock.updateToken.mockRejectedValueOnce(new Error('expired'));
      await keycloakMock.onTokenExpired?.();
      await Promise.resolve();

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(keycloakMock.login).toHaveBeenCalledOnce();
    });

    it('marks initialization complete without authenticating when keycloak denies', async () => {
      keycloakMock.init.mockResolvedValueOnce(false);

      await useAuthStore.getState().initialize();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
      expect(state.initialized).toBe(true);
    });

    it('degrades gracefully to unauthenticated when init throws', async () => {
      keycloakMock.init.mockRejectedValueOnce(new Error('network down'));

      await useAuthStore.getState().initialize();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(false);
      expect(state.initialized).toBe(true);
    });
  });
});
