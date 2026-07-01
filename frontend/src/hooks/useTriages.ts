import { useCallback, useEffect, useState } from 'react';
import { listTriages, type ListTriagesParams } from '../api/triages';
import { cacheTriageList, getCachedTriages } from '../offline/triageOffline';
import { isNetworkError } from '../api/errors';
import type { TriageListItem, Pagination } from '../types';

export function useTriages(initialParams: ListTriagesParams = {}) {
  const [params, setParams] = useState<ListTriagesParams>({
    page: 1,
    per_page: 20,
    ...initialParams,
  });
  const [triages, setTriages] = useState<TriageListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setIsLoading(true);
    setError(null);
    listTriages(params)
      .then((res) => {
        if (cancelled) return;
        setTriages(res.data);
        setPagination(res.pagination);
        void cacheTriageList(res.data);
      })
      .catch(async (err: unknown) => {
        if (cancelled) return;
        // Offline / network failure: serve the local cache (which includes
        // pending offline-created records) instead of an empty error state.
        if (!navigator.onLine || isNetworkError(err)) {
          const cached = await getCachedTriages();
          setTriages(cached);
          setPagination((p) => ({ ...p, total: cached.length }));
          setError(null);
        } else {
          setError(err instanceof Error ? err.message : 'Falha ao carregar triagens');
        }
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [params]);

  const goToPage = useCallback((page: number) => {
    setParams((p) => ({ ...p, page }));
  }, []);

  const filterByPerson = useCallback((personID: string | undefined) => {
    setParams((p) => ({ ...p, person_id: personID, page: 1 }));
  }, []);

  return { triages, pagination, isLoading, error, goToPage, filterByPerson };
}
