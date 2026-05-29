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
    if (!params) {
      setState({ report: null, isLoading: false, error: null });
      return;
    }
    let cancelled = false;
    setState({ report: null, isLoading: true, error: null });
    getAttendanceReport(params)
      .then((report) => {
        if (cancelled) return;
        setState({ report, isLoading: false, error: null });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setState({
          report: null,
          isLoading: false,
          error: err instanceof Error ? err.message : 'Falha ao gerar relatório',
        });
      });
    return () => {
      cancelled = true;
    };
  }, [params]);

  const generate = useCallback((next: AttendanceReportParams) => {
    setParams(next);
  }, []);

  return { ...state, params, generate };
}
