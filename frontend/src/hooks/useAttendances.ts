import { useCallback, useEffect, useState } from 'react';
import { listAttendances, type ListAttendancesParams } from '../api/attendances';
import type { AttendanceListItem, AttendanceStatus, Pagination } from '../types';

export function useAttendances(initialParams: ListAttendancesParams = {}) {
  const [params, setParams] = useState<ListAttendancesParams>({
    page: 1,
    per_page: 20,
    ...initialParams,
  });
  const [attendances, setAttendances] = useState<AttendanceListItem[]>([]);
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
    listAttendances(params)
      .then((res) => {
        if (cancelled) return;
        setAttendances(res.data);
        setPagination(res.pagination);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Falha ao carregar atendimentos');
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

  const filterByStatus = useCallback((status: AttendanceStatus | undefined) => {
    setParams((p) => ({ ...p, status, page: 1 }));
  }, []);

  return { attendances, pagination, isLoading, error, goToPage, filterByStatus, params };
}
