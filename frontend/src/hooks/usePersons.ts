import { useCallback, useEffect, useRef, useState } from 'react';
import { listPersons } from '../api/persons';
import { cachePersonList, getCachedPersons } from '../offline/personOffline';
import { isNetworkError } from '../api/errors';
import type { Pagination, PersonListItem } from '../types';

type PersonsResult =
  | { kind: 'ok'; persons: PersonListItem[]; pagination: Pagination }
  | { kind: 'cached'; persons: PersonListItem[] }
  | { kind: 'error'; message: string };

async function fetchPersonsResult(
  q: string,
  page: number,
  agreementFilter: string,
): Promise<PersonsResult> {
  try {
    const result = await listPersons({
      q: q || undefined,
      page,
      agreement_status: agreementFilter || undefined,
    });
    void cachePersonList(result.data);
    return { kind: 'ok', persons: result.data, pagination: result.pagination };
  } catch (err) {
    // Offline / network failure: serve the local cache (which includes
    // pending offline-created records) instead of an empty error state.
    if (!navigator.onLine || isNetworkError(err)) {
      return { kind: 'cached', persons: await getCachedPersons() };
    }
    return {
      kind: 'error',
      message: err instanceof Error ? err.message : 'Erro ao carregar pessoas',
    };
  }
}

function usePersonsData() {
  const [persons, setPersons] = useState<PersonListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPersons = useCallback(
    async (q: string, page: number, agreementFilter: string) => {
      setIsLoading(true);
      setError(null);
      const result = await fetchPersonsResult(q, page, agreementFilter);
      if (result.kind === 'ok') {
        setPersons(result.persons);
        setPagination(result.pagination);
      } else if (result.kind === 'cached') {
        setPersons(result.persons);
        setPagination((p) => ({ ...p, total: result.persons.length }));
      } else {
        setError(result.message);
        setPersons([]);
      }
      setIsLoading(false);
    },
    [],
  );

  return { persons, pagination, isLoading, error, fetchPersons };
}

export function usePersons() {
  const { persons, pagination, isLoading, error, fetchPersons } =
    usePersonsData();
  const [query, setQuery] = useState('');
  const [agreementStatus, setAgreementStatus] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const search = useCallback(
    (q: string) => {
      setQuery(q);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        fetchPersons(q, 1, agreementStatus);
      }, 300);
    },
    [fetchPersons, agreementStatus],
  );

  const filterByAgreement = useCallback(
    (status: string) => {
      setAgreementStatus(status);
      fetchPersons(query, 1, status);
    },
    [fetchPersons, query],
  );

  const goToPage = useCallback(
    (page: number) => {
      fetchPersons(query, page, agreementStatus);
    },
    [fetchPersons, query, agreementStatus],
  );

  const refresh = useCallback(() => {
    fetchPersons(query, pagination.page, agreementStatus);
  }, [fetchPersons, query, pagination.page, agreementStatus]);

  useEffect(() => {
    void (async () => {
      await fetchPersons('', 1, '');
    })();
  }, [fetchPersons]);

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  return {
    persons,
    pagination,
    isLoading,
    error,
    query,
    agreementStatus,
    search,
    filterByAgreement,
    goToPage,
    refresh,
  };
}
