import { useCallback, useEffect, useRef, useState } from 'react';
import { db } from '../offline/db';
import {
  drainQueue,
  mergePullRecords,
  getRetryable,
  backoffDelay,
} from '../offline/syncEngine';
import { syncPush, syncPull } from '../api/sync';

const LAST_SYNC_KEY = 'lastSyncAt';
const EPOCH = '1970-01-01T00:00:00Z';

export interface UseOnlineSyncOptions {
  /**
   * Overrides the retry delay computation (defaults to the exponential
   * backoffDelay schedule). Primarily a test seam to avoid multi-minute waits.
   */
  retryDelayMs?: (retryCount: number) => number;
  /** When true (default), the hook drains automatically on reconnect. */
  autoStart?: boolean;
}

export interface OnlineSyncState {
  isOnline: boolean;
  isSyncing: boolean;
  pendingCount: number;
  conflictCount: number;
  lastError: string | null;
  syncNow: () => Promise<void>;
}

async function countByStatus(status: 'pending' | 'conflict'): Promise<number> {
  const tables = [db.persons, db.triages, db.attendances];
  const counts = await Promise.all(
    tables.map((t) => t.where('syncStatus').equals(status).count()),
  );
  return counts.reduce((a, b) => a + b, 0);
}

async function getCursor(): Promise<string> {
  const meta = await db.syncMeta.get(LAST_SYNC_KEY);
  return meta?.value ?? EPOCH;
}

async function setCursor(value: string): Promise<void> {
  await db.syncMeta.put({ key: LAST_SYNC_KEY, value });
}

/**
 * useOnlineSync drains the offline queue and pulls server updates. It exposes
 * the pending and conflict counts for the UI, a manual `syncNow`, and (when
 * autoStart is on) drains automatically whenever the browser regains
 * connectivity. The drainer is a no-op while offline.
 */
export function useOnlineSync(options: UseOnlineSyncOptions = {}): OnlineSyncState {
  const { autoStart = true, retryDelayMs = backoffDelay } = options;
  const [isOnline, setIsOnline] = useState(() => navigator.onLine);
  const [isSyncing, setIsSyncing] = useState(false);
  const [pendingCount, setPendingCount] = useState(0);
  const [conflictCount, setConflictCount] = useState(0);
  const [lastError, setLastError] = useState<string | null>(null);
  const retryTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const refreshCounts = useCallback(async () => {
    // Pending = items still eligible for an automatic drain. Conflicted and
    // dead-lettered items are excluded (they await operator resolution, not
    // retry) so the count can actually reach zero once everything has drained.
    const retryable = await getRetryable();
    setPendingCount(retryable.length);
    setConflictCount(await countByStatus('conflict'));
  }, []);

  // syncImplRef breaks the syncNow ↔ scheduleRetry cycle: scheduleRetry fires
  // the latest syncNow via the ref without taking it as a dependency.
  const syncImplRef = useRef<() => Promise<void>>(async () => {});

  const scheduleRetry = useCallback(async () => {
    if (retryTimer.current) clearTimeout(retryTimer.current);
    if (!navigator.onLine) return;
    // Only items that have already failed at least once warrant a backoff retry.
    const failed = (await getRetryable()).filter((item) => item.retryCount > 0);
    if (failed.length === 0) return;
    const minRetries = Math.min(...failed.map((item) => item.retryCount));
    retryTimer.current = setTimeout(() => {
      void syncImplRef.current();
    }, retryDelayMs(minRetries));
  }, [retryDelayMs]);

  const syncNow = useCallback(async () => {
    if (!navigator.onLine) return;
    setIsSyncing(true);
    setLastError(null);
    try {
      await drainQueue(syncPush);
      const since = await getCursor();
      const pull = await syncPull(since);
      if (pull.records.length > 0) {
        await mergePullRecords(pull.records);
      }
      await setCursor(pull.next_since ?? pull.server_timestamp);
    } catch (err) {
      setLastError(err instanceof Error ? err.message : String(err));
    } finally {
      await refreshCounts();
      setIsSyncing(false);
      if (autoStart) await scheduleRetry();
    }
  }, [refreshCounts, autoStart, scheduleRetry]);

  useEffect(() => {
    syncImplRef.current = syncNow;
  }, [syncNow]);

  useEffect(() => {
    void (async () => {
      await refreshCounts();
    })();
  }, [refreshCounts]);

  useEffect(() => {
    function handleOnline() {
      setIsOnline(true);
      if (autoStart) void syncNow();
    }
    function handleOffline() {
      setIsOnline(false);
    }
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [autoStart, syncNow]);

  useEffect(() => {
    return () => {
      if (retryTimer.current) clearTimeout(retryTimer.current);
    };
  }, []);

  return { isOnline, isSyncing, pendingCount, conflictCount, lastError, syncNow };
}
