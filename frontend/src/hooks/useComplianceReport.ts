import { useCallback, useEffect, useState } from 'react';
import { getComplianceReport } from '../api/compliance';
import type { ComplianceReport, ComplianceReportParams } from '../types/compliance';

interface UseComplianceReportState {
  report: ComplianceReport | null;
  isLoading: boolean;
  error: string | null;
}

export function useComplianceReport(initial?: ComplianceReportParams) {
  const [params, setParams] = useState<ComplianceReportParams | null>(
    initial ?? null,
  );
  const [state, setState] = useState<UseComplianceReportState>({
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
    getComplianceReport(params)
      .then((report) => {
        if (cancelled) return;
        setState({ report, isLoading: false, error: null });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setState({
          report: null,
          isLoading: false,
          error:
            err instanceof Error ? err.message : 'Falha ao gerar relatório',
        });
      });
    return () => {
      cancelled = true;
    };
  }, [params]);

  const generate = useCallback((next: ComplianceReportParams) => {
    setParams(next);
  }, []);

  return { ...state, params, generate };
}
