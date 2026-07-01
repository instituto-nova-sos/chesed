import { useCallback, useEffect, useState } from 'react';
import { getConflicts, discardConflict, requeueConflict } from '../offline/syncEngine';
import type { SyncQueueItem } from '../offline/db';

export interface UseSyncConflicts {
  conflicts: SyncQueueItem[];
  isLoading: boolean;
  refresh: () => Promise<void>;
  discard: (queueId: number) => Promise<void>;
  resubmit: (queueId: number) => Promise<void>;
}

/**
 * useSyncConflicts exposes the queued records the server rejected as conflicts,
 * plus the two operator actions: discard (drop the local record) and resubmit
 * (clear the conflict so the drainer retries it, last-write-wins). Both refresh
 * the list so the UI reflects the queue immediately.
 */
export function useSyncConflicts(): UseSyncConflicts {
  const [conflicts, setConflicts] = useState<SyncQueueItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(async () => {
    setIsLoading(true);
    try {
      setConflicts(await getConflicts());
    } finally {
      setIsLoading(false);
    }
  }, []);

  const discard = useCallback(
    async (queueId: number) => {
      await discardConflict(queueId);
      await refresh();
    },
    [refresh],
  );

  const resubmit = useCallback(
    async (queueId: number) => {
      await requeueConflict(queueId);
      await refresh();
    },
    [refresh],
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { conflicts, isLoading, refresh, discard, resubmit };
}
