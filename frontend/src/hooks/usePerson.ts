import { useCallback, useEffect, useState } from 'react';
import { getPerson } from '../api/persons';
import type { PersonDetail } from '../types';

export function usePerson(id: string | undefined) {
  const [person, setPerson] = useState<PersonDetail | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPerson = useCallback(async (personId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await getPerson(personId);
      setPerson(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao carregar pessoa');
      setPerson(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!id) return;
    void (async () => {
      await fetchPerson(id);
    })();
  }, [id, fetchPerson]);

  const refresh = useCallback(() => {
    if (id) fetchPerson(id);
  }, [id, fetchPerson]);

  return { person, isLoading, error, refresh };
}
