import { useCallback, useEffect, useRef, useState } from 'react';
import { listPersonDocuments } from '../api/documents';
import type { PersonDocument } from '../types';

/**
 * Loads a person's documents. Pass `undefined` to skip fetching entirely —
 * SECRETARY users may upload but not list (docs/16), so the section disables
 * the fetch instead of triggering a guaranteed 403.
 */
export function usePersonDocuments(personId: string | undefined) {
  const [documents, setDocuments] = useState<PersonDocument[]>([]);
  const [isLoading, setIsLoading] = useState(Boolean(personId));
  const [error, setError] = useState<string | null>(null);

  // Guards a stale in-flight response from overwriting a newer one when the
  // person id changes quickly.
  const requestSeq = useRef(0);

  // State transitions live inside promise continuations only, so the effect
  // never sets state synchronously (react-hooks/set-state-in-effect).
  const refresh = useCallback((): Promise<void> => {
    if (!personId) return Promise.resolve();
    requestSeq.current += 1;
    const seq = requestSeq.current;
    return listPersonDocuments(personId)
      .then((response) => {
        if (seq !== requestSeq.current) return;
        setDocuments(response.data);
        setError(null);
      })
      .catch(() => {
        if (seq !== requestSeq.current) return;
        setError('Não foi possível carregar os documentos.');
      })
      .finally(() => {
        if (seq === requestSeq.current) setIsLoading(false);
      });
  }, [personId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { documents, isLoading, error, refresh };
}
