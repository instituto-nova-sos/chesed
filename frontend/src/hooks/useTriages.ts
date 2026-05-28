import { useCallback, useEffect, useState } from 'react';
import { listTriages, type ListTriagesParams } from '../api/triages';
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
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Falha ao carregar triagens');
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
