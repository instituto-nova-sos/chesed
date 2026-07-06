import { useCallback, useEffect, useState } from 'react';
import { listAuditLogs } from '../api/audit';
import type { AuditLogFilters, AuditLogPage } from '../types/audit';

interface UseAuditLogsState {
  page: AuditLogPage | null;
  isLoading: boolean;
  error: string | null;
}

export function useAuditLogs(initial?: AuditLogFilters) {
  const [filters, setFilters] = useState<AuditLogFilters | null>(initial ?? null);
  const [state, setState] = useState<UseAuditLogsState>({
    page: null,
    isLoading: false,
    error: null,
  });

  useEffect(() => {
    if (!filters) {
      setState({ page: null, isLoading: false, error: null });
      return;
    }
    let cancelled = false;
    setState({ page: null, isLoading: true, error: null });
    listAuditLogs(filters)
      .then((page) => {
        if (cancelled) return;
        setState({ page, isLoading: false, error: null });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setState({
          page: null,
          isLoading: false,
          error:
            err instanceof Error ? err.message : 'Falha ao carregar registros',
        });
      });
    return () => {
      cancelled = true;
    };
  }, [filters]);

  const query = useCallback((next: AuditLogFilters) => {
    setFilters(next);
  }, []);

  return { ...state, filters, query };
}
