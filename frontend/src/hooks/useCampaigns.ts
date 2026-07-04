import { useCallback, useEffect, useState } from 'react';
import { listCampaigns, type ListCampaignsParams } from '../api/campaigns';
import type { CampaignListItem, Pagination } from '../types';

// Campaign management is an online-only surface (docs/12): offline reads are
// served by the service worker's NetworkFirst cache, not an IndexedDB layer,
// so this hook has no offline fallback on purpose. Loading/error transitions
// happen in event handlers and promise continuations (never synchronously
// inside the effect) to avoid cascading-render re-entry.
export function useCampaigns(initialParams: ListCampaignsParams = {}) {
  const [params, setParams] = useState<ListCampaignsParams>({
    page: 1,
    per_page: 20,
    ...initialParams,
  });
  const [campaigns, setCampaigns] = useState<CampaignListItem[]>([]);
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
    listCampaigns(params)
      .then((res) => {
        if (cancelled) return;
        setCampaigns(res.data);
        setPagination(res.pagination);
        setError(null);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setCampaigns([]);
        setError(err instanceof Error ? err.message : 'Falha ao carregar campanhas');
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

  const filterByStatus = useCallback((status: string | undefined) => {
    setIsLoading(true);
    setParams((p) => ({ ...p, status: status || undefined, page: 1 }));
  }, []);

  return { campaigns, pagination, isLoading, error, goToPage, filterByStatus };
}

const LINKABLE_CAMPAIGN_STATUSES = new Set(['PLANNED', 'ACTIVE']);

// Loads the campus campaigns a triage/attendance can be linked to. Campaigns
// are campus-scoped and few, so a single page covers them; the link is
// optional context, so a fetch failure degrades to an empty list instead of
// surfacing an error that would block the create form.
export function useLinkableCampaigns(): CampaignListItem[] {
  const [campaigns, setCampaigns] = useState<CampaignListItem[]>([]);

  useEffect(() => {
    let cancelled = false;
    listCampaigns({ per_page: 100 })
      .then((res) => {
        if (cancelled) return;
        setCampaigns(res.data.filter((c) => LINKABLE_CAMPAIGN_STATUSES.has(c.status)));
      })
      .catch(() => {
        // Selector simply stays empty; creating without a campaign is valid.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return campaigns;
}
