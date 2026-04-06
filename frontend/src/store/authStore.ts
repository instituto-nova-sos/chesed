import { create } from 'zustand';
import { keycloak } from '../auth/keycloak';

interface KeycloakTokenParsed {
  email?: string;
  realm_access?: { roles: string[] };
  campus_id?: string;
  person_id?: string;
}

interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  initialized: boolean;
  email: string | null;
  roles: string[];
  campusId: string | null;
  personId: string | null;
  token: string | null;
  initialize: () => Promise<void>;
  logout: () => void;
  getToken: () => Promise<string | null>;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: true,
  initialized: false,
  email: null,
  roles: [],
  campusId: null,
  personId: null,
  token: null,

  initialize: async () => {
    try {
      const authenticated = await keycloak.init({
        onLoad: 'login-required',
        pkceMethod: 'S256',
        checkLoginIframe: false,
      });

      if (authenticated) {
        const parsed = keycloak.tokenParsed as KeycloakTokenParsed | undefined;
        set({
          isAuthenticated: true,
          isLoading: false,
          initialized: true,
          email: parsed?.email ?? null,
          roles: parsed?.realm_access?.roles ?? [],
          campusId: parsed?.campus_id ?? null,
          personId: parsed?.person_id ?? null,
          token: keycloak.token ?? null,
        });

        keycloak.onTokenExpired = () => {
          keycloak
            .updateToken(30)
            .then((refreshed) => {
              if (refreshed) {
                set({ token: keycloak.token ?? null });
              }
            })
            .catch(() => {
              set({ isAuthenticated: false });
              keycloak.login();
            });
        };
      } else {
        set({ isLoading: false, initialized: true });
      }
    } catch {
      set({ isLoading: false, initialized: true, isAuthenticated: false });
    }
  },

  logout: () => {
    keycloak.logout({ redirectUri: window.location.origin });
  },

  getToken: async () => {
    try {
      await keycloak.updateToken(30);
      const token = keycloak.token ?? null;
      set({ token });
      return token;
    } catch {
      set({ isAuthenticated: false });
      keycloak.login();
      return null;
    }
  },
}));
