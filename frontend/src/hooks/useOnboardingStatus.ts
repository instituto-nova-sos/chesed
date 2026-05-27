import { useState, useEffect, useCallback } from 'react';
import { getOnboardingStatus, type OnboardingStatus } from '../api/auth';

export function useOnboardingStatus() {
  const [status, setStatus] = useState<OnboardingStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetch = useCallback(() => {
    setLoading(true);
    setError(null);
    getOnboardingStatus()
      .then(setStatus)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err : new Error('Failed to fetch onboarding status'));
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetch();
  }, [fetch]);

  return { status, loading, error, refetch: fetch };
}
