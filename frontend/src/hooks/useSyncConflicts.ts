import { useCallback, useEffect, useState } from 'react';
import {
  getConflicts,
  getConflictSnapshot,
  discardConflict,
  requeueConflict,
  resolveKeepLocal,
  resolveKeepServer,
  resolveMerged,
} from '../offline/syncEngine';
import type { ConflictSnapshot, SyncQueueItem } from '../offline/db';

export interface UseSyncConflicts {
  conflicts: SyncQueueItem[];
  isLoading: boolean;
  refresh: () => Promise<void>;
  discard: (queueId: number) => Promise<void>;
  resubmit: (queueId: number) => Promise<void>;
  snapshotFor: (entityId: string) => Promise<ConflictSnapshot | undefined>;
  keepLocal: (queueId: number) => Promise<void>;
  keepServer: (queueId: number) => Promise<void>;
  merge: (queueId: number, mergedData: Record<string, unknown>) => Promise<void>;
}

/**
 * useSyncConflicts exposes the queued records the server rejected as conflicts,
 * plus the operator actions: discard (drop the local record), resubmit
 * (last-write-wins retry), and the field-level resolutions keepLocal / keepServer
 * / merge (S12.2). `snapshotFor` reads the captured server version for a conflict.
 * Every mutating action refreshes the list so the UI reflects the queue
 * immediately.
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

  const snapshotFor = useCallback(
    (entityId: string) => getConflictSnapshot(entityId),
    [],
  );

  const keepLocal = useCallback(
    async (queueId: number) => {
      await resolveKeepLocal(queueId);
      await refresh();
    },
    [refresh],
  );

  const keepServer = useCallback(
    async (queueId: number) => {
      await resolveKeepServer(queueId);
      await refresh();
    },
    [refresh],
  );

  const merge = useCallback(
    async (queueId: number, mergedData: Record<string, unknown>) => {
      await resolveMerged(queueId, mergedData);
      await refresh();
    },
    [refresh],
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return {
    conflicts,
    isLoading,
    refresh,
    discard,
    resubmit,
    snapshotFor,
    keepLocal,
    keepServer,
    merge,
  };
}
