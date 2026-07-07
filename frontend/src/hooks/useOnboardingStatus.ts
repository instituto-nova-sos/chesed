import { useState, useEffect, useCallback } from 'react';
import { getOnboardingStatus, type OnboardingStatus } from '../api/auth';

export function useOnboardingStatus() {
  const [status, setStatus] = useState<OnboardingStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setStatus(await getOnboardingStatus());
    } catch (err: unknown) {
      setError(err instanceof Error ? err : new Error('Failed to fetch onboarding status'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void (async () => {
      await fetch();
    })();
  }, [fetch]);

  return { status, loading, error, refetch: fetch };
}
