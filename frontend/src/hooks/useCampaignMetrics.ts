import { useEffect, useState } from 'react';
import { getCampaignMetrics } from '../api/campaigns';
import { isNetworkError } from '../api/errors';
import type { CampaignMetrics } from '../types';

// Metrics are an online-only read surface (E06 policy): when offline the hook
// reports a clear message instead of serving stale numbers (S07.4 criterion).
// `id` and `enabled` are stable per page mount, so loading state is seeded
// from them and only transitions inside promise continuations.
export function useCampaignMetrics(id: string | undefined, enabled: boolean) {
  const [metrics, setMetrics] = useState<CampaignMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(id) && enabled);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id || !enabled) return;
    let cancelled = false;
    getCampaignMetrics(id)
      .then((m) => {
        if (cancelled) return;
        setMetrics(m);
        setError(null);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        if (!navigator.onLine || isNetworkError(err)) {
          setError('Indicadores indisponíveis offline.');
        } else {
          setError('Falha ao carregar indicadores.');
        }
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, enabled]);

  return { metrics, isLoading, error };
}
