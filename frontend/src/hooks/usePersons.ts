import { useCallback, useEffect, useRef, useState } from 'react';
import { listPersons } from '../api/persons';
import type { Pagination, PersonListItem } from '../types';

export function usePersons() {
  const [persons, setPersons] = useState<PersonListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [agreementStatus, setAgreementStatus] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const fetchPersons = useCallback(
    async (q: string, page: number, agreementFilter: string) => {
      setIsLoading(true);
      setError(null);
      try {
        const result = await listPersons({
          q: q || undefined,
          page,
          agreement_status: agreementFilter || undefined,
        });
        setPersons(result.data);
        setPagination(result.pagination);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Erro ao carregar pessoas');
        setPersons([]);
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

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
    fetchPersons('', 1, '');
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
