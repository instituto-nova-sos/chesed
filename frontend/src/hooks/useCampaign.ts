import { useCallback, useEffect, useRef, useState } from 'react';
import {
  getCampaign,
  addTeamMember as apiAddTeamMember,
  removeTeamMember as apiRemoveTeamMember,
} from '../api/campaigns';
import type { AddTeamMemberInput, CampaignDetail } from '../types';

export function useCampaign(id: string | undefined) {
  const [campaign, setCampaign] = useState<CampaignDetail | null>(null);
  const [isLoading, setIsLoading] = useState(Boolean(id));
  const [error, setError] = useState<string | null>(null);

  // Guards a stale in-flight response from overwriting a newer one when the
  // id changes quickly (review N2).
  const requestSeq = useRef(0);

  // State transitions live inside promise continuations only, so the effect
  // never sets state synchronously (react-hooks/set-state-in-effect).
  const reload = useCallback((): Promise<void> => {
    if (!id) return Promise.resolve();
    requestSeq.current += 1;
    const seq = requestSeq.current;
    return getCampaign(id)
      .then((detail) => {
        if (seq !== requestSeq.current) return;
        setCampaign(detail);
        setError(null);
      })
      .catch((err: unknown) => {
        if (seq !== requestSeq.current) return;
        setError(err instanceof Error ? err.message : 'Falha ao carregar campanha');
      })
      .finally(() => {
        if (seq === requestSeq.current) setIsLoading(false);
      });
  }, [id]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Team mutations rethrow so the page can map API errors (409 duplicate,
  // 400 validation) into readable messages; the detail is reloaded on success.
  const addMember = useCallback(
    async (input: AddTeamMemberInput) => {
      if (!id) return;
      await apiAddTeamMember(id, input);
      await reload();
    },
    [id, reload],
  );

  const removeMember = useCallback(
    async (personId: string) => {
      if (!id) return;
      await apiRemoveTeamMember(id, personId);
      await reload();
    },
    [id, reload],
  );

  return { campaign, isLoading, error, addMember, removeMember, reload };
}
