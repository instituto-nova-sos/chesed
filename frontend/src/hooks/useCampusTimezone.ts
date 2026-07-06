import { useAuthStore } from '../store/authStore';

/**
 * Returns the current user's campus IANA time zone, or `undefined` when it is
 * not yet available in context. Callers pass the result to `formatDateTime`,
 * which falls back to the browser zone when the value is `undefined`.
 *
 * The value is sourced from the auth store's `campusTimezone`, which is
 * expected to be populated from the `/auth/me` (onboarding status) response
 * once the backend exposes a campus `timezone` field.
 */
export function useCampusTimezone(): string | undefined {
  const timezone = useAuthStore((s) => s.campusTimezone);
  return timezone ?? undefined;
}
