import { useCallback, useEffect, useState } from 'react';
import { getTriage } from '../api/triages';
import type { Triage } from '../types';

export function useTriage(id: string | undefined) {
  const [triage, setTriage] = useState<Triage | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(id));
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(() => {
    if (!id) return;
    setIsLoading(true);
    setError(null);
    getTriage(id)
      .then(setTriage)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Falha ao carregar triagem');
      })
      .finally(() => setIsLoading(false));
  }, [id]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return { triage, isLoading, error, refetch };
}
