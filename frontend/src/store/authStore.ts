import { create } from 'zustand';
import { keycloak } from '../auth/keycloak';

interface KeycloakTokenParsed {
  email?: string;
  email_verified?: boolean;
  realm_access?: { roles: string[] };
  person_id?: string;
}

interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  initialized: boolean;
  email: string | null;
  emailVerified: boolean;
  roles: string[];
  campusId: string | null;
  campusTimezone: string | null;
  personId: string | null;
  token: string | null;
  initialize: () => Promise<void>;
  logout: () => void;
  getToken: () => Promise<string | null>;
  setCampusId: (campusId: string) => void;
  setCampusTimezone: (timezone: string | null) => void;
}

type SetAuthState = (partial: Partial<AuthState>) => void;

function authenticatedState(parsed: KeycloakTokenParsed | undefined): Partial<AuthState> {
  const {
    email = null,
    email_verified: emailVerified = false,
    realm_access: realmAccess,
    person_id: personId = null,
  } = parsed ?? {};
  return {
    isAuthenticated: true,
    isLoading: false,
    initialized: true,
    email,
    emailVerified,
    roles: realmAccess?.roles ?? [],
    campusId: null,
    campusTimezone: null,
    personId,
    token: keycloak.token ?? null,
  };
}

function installTokenRefresh(set: SetAuthState): void {
  keycloak.onTokenExpired = () => {
    keycloak
      .updateToken(30)
      .then((refreshed) => {
        if (refreshed) set({ token: keycloak.token ?? null });
      })
      .catch(() => {
        set({ isAuthenticated: false });
        keycloak.login();
      });
  };
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: true,
  initialized: false,
  email: null,
  emailVerified: false,
  roles: [],
  campusId: null,
  campusTimezone: null,
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
        set(authenticatedState(parsed));
        installTokenRefresh(set);
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

  setCampusId: (campusId: string) => {
    set({ campusId });
  },

  setCampusTimezone: (timezone: string | null) => {
    set({ campusTimezone: timezone });
  },
}));
