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
    const activeFilters = filters;
    let cancelled = false;
    async function load() {
      if (!activeFilters) {
        setState({ page: null, isLoading: false, error: null });
        return;
      }
      setState({ page: null, isLoading: true, error: null });
      try {
        const page = await listAuditLogs(activeFilters);
        if (cancelled) return;
        setState({ page, isLoading: false, error: null });
      } catch (err: unknown) {
        if (cancelled) return;
        setState({
          page: null,
          isLoading: false,
          error:
            err instanceof Error ? err.message : 'Falha ao carregar registros',
        });
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [filters]);

  const query = useCallback((next: AuditLogFilters) => {
    setFilters(next);
  }, []);

  return { ...state, filters, query };
}
