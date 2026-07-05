import { useCallback, useEffect, useRef, useState } from 'react';
import { getDonation } from '../api/donations';
import type { DonationDetail } from '../types';

export function useDonation(id: string | undefined) {
  const [donation, setDonation] = useState<DonationDetail | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(id));
  const [error, setError] = useState<string | null>(null);

  // Guards a stale in-flight response from overwriting a newer one when the
  // id changes quickly.
  const requestSeq = useRef(0);

  // State transitions live inside promise continuations only, so the effect
  // never sets state synchronously (react-hooks/set-state-in-effect).
  const reload = useCallback((): Promise<void> => {
    if (!id) return Promise.resolve();
    requestSeq.current += 1;
    const seq = requestSeq.current;
    return getDonation(id)
      .then((detail) => {
        if (seq !== requestSeq.current) return;
        setDonation(detail);
        setError(null);
      })
      .catch((err: unknown) => {
        if (seq !== requestSeq.current) return;
        setError(err instanceof Error ? err.message : 'Falha ao carregar doação');
      })
      .finally(() => {
        if (seq === requestSeq.current) setIsLoading(false);
      });
  }, [id]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { donation, isLoading, error, reload };
}
