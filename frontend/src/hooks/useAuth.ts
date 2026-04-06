import { useAuthStore } from '../store/authStore';

const ROLE_HIERARCHY: Record<string, number> = {
  VOLUNTEER: 1,
  SECRETARY: 2,
  PROFESSIONAL: 3,
  COORDINATOR: 4,
  ADMIN: 5,
};

export function useAuth() {
  const store = useAuthStore();

  const hasRole = (...requiredRoles: string[]): boolean => {
    return store.roles.some((userRole) => requiredRoles.includes(userRole));
  };

  const hasMinRole = (minRole: string): boolean => {
    const minLevel = ROLE_HIERARCHY[minRole] ?? 0;
    return store.roles.some((role) => (ROLE_HIERARCHY[role] ?? 0) >= minLevel);
  };

  return {
    isAuthenticated: store.isAuthenticated,
    isLoading: store.isLoading,
    initialized: store.initialized,
    email: store.email,
    roles: store.roles,
    campusId: store.campusId,
    personId: store.personId,
    logout: store.logout,
    hasRole,
    hasMinRole,
  };
}
