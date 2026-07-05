import { useCallback, useEffect, useState } from 'react';
import { listDonations, type ListDonationsParams } from '../api/donations';
import type { DonationListItem, Pagination } from '../types';

// Donation tracking is an online-only surface (docs/12): offline reads are
// served by the service worker's NetworkFirst cache, not an IndexedDB layer,
// so this hook has no offline fallback on purpose. Loading/error transitions
// happen in event handlers and promise continuations (never synchronously
// inside the effect) to avoid cascading-render re-entry.
export function useDonations(initialParams: ListDonationsParams = {}) {
  const [params, setParams] = useState<ListDonationsParams>({
    page: 1,
    per_page: 20,
    ...initialParams,
  });
  const [donations, setDonations] = useState<DonationListItem[]>([]);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    per_page: 20,
    total: 0,
    total_pages: 0,
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listDonations(params)
      .then((res) => {
        if (cancelled) return;
        setDonations(res.data);
        setPagination(res.pagination);
        setError(null);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setDonations([]);
        setError(err instanceof Error ? err.message : 'Falha ao carregar doações');
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [params]);

  const goToPage = useCallback((page: number) => {
    setIsLoading(true);
    setParams((p) => ({ ...p, page }));
  }, []);

  const filterByType = useCallback((donationType: string | undefined) => {
    setIsLoading(true);
    setParams((p) => ({ ...p, donation_type: donationType || undefined, page: 1 }));
  }, []);

  return { donations, pagination, isLoading, error, goToPage, filterByType };
}
