import { useCallback, useEffect, useState } from 'react';
import { getAttendanceReport } from '../api/reports';
import type { AttendanceReport, AttendanceReportParams } from '../types';

interface UseAttendanceReportState {
  report: AttendanceReport | null;
  isLoading: boolean;
  error: string | null;
}

export function useAttendanceReport(initial?: AttendanceReportParams) {
  const [params, setParams] = useState<AttendanceReportParams | null>(
    initial ?? null,
  );
  const [state, setState] = useState<UseAttendanceReportState>({
    report: null,
    isLoading: false,
    error: null,
  });

  useEffect(() => {
    const activeParams = params;
    let cancelled = false;
    async function load() {
      if (!activeParams) {
        setState({ report: null, isLoading: false, error: null });
        return;
      }
      setState({ report: null, isLoading: true, error: null });
      try {
        const report = await getAttendanceReport(activeParams);
        if (cancelled) return;
        setState({ report, isLoading: false, error: null });
      } catch (err: unknown) {
        if (cancelled) return;
        setState({
          report: null,
          isLoading: false,
          error: err instanceof Error ? err.message : 'Falha ao gerar relatório',
        });
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [params]);

  const generate = useCallback((next: AttendanceReportParams) => {
    setParams(next);
  }, []);

  return { ...state, params, generate };
}
