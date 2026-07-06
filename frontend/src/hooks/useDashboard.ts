import { useEffect, useState } from 'react';
import { getDashboard } from '../api/reports';
import type { DashboardMetrics } from '../types';

interface UseDashboardState {
  data: DashboardMetrics | null;
  isLoading: boolean;
  error: string | null;
}

export function useDashboard() {
  const [state, setState] = useState<UseDashboardState>({
    data: null,
    isLoading: true,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;
    getDashboard()
      .then((data) => {
        if (cancelled) return;
        setState({ data, isLoading: false, error: null });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setState({
          data: null,
          isLoading: false,
          error: err instanceof Error ? err.message : 'Falha ao carregar painel',
        });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return state;
}
