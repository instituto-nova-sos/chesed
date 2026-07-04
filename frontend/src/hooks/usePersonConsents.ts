import { useCallback, useEffect, useRef, useState } from 'react';
import { listPersonConsents, revokeConsent as apiRevokeConsent } from '../api/consents';
import type { Consent } from '../types/consent';

export function usePersonConsents(personId: string | undefined) {
  const [consents, setConsents] = useState<Consent[]>([]);
  const [isLoading, setIsLoading] = useState(Boolean(personId));
  const [error, setError] = useState<string | null>(null);

  // Guards a stale in-flight response from overwriting a newer one when the
  // person id changes quickly (same pattern as useCampaign).
  const requestSeq = useRef(0);

  const reload = useCallback((): Promise<void> => {
    if (!personId) return Promise.resolve();
    requestSeq.current += 1;
    const seq = requestSeq.current;
    return listPersonConsents(personId)
      .then((result) => {
        if (seq !== requestSeq.current) return;
        setConsents(result.data);
        setError(null);
      })
      .catch(() => {
        if (seq !== requestSeq.current) return;
        setError('Erro ao carregar consentimentos');
      })
      .finally(() => {
        if (seq === requestSeq.current) setIsLoading(false);
      });
  }, [personId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Rethrows so the caller can map API errors into readable messages; the
  // registry is reloaded on success.
  const revoke = useCallback(
    async (consentId: string, reason: string) => {
      await apiRevokeConsent(consentId, reason);
      await reload();
    },
    [reload],
  );

  return { consents, isLoading, error, reload, revoke };
}
