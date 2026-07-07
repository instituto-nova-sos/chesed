import { useCallback, useEffect, useState } from 'react';
import { getAttendance } from '../api/attendances';
import type { AttendanceDetail } from '../types';

export function useAttendance(id: string | undefined) {
  const [attendance, setAttendance] = useState<AttendanceDetail | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(id));
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    if (!id) return;
    setIsLoading(true);
    setError(null);
    try {
      setAttendance(await getAttendance(id));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Falha ao carregar atendimento');
    } finally {
      setIsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void (async () => {
      await refetch();
    })();
  }, [refetch]);

  return { attendance, isLoading, error, refetch };
}
