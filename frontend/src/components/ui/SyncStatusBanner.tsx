import { useOnlineSync } from '../../hooks/useOnlineSync';

/**
 * SyncStatusBanner surfaces the offline-sync state: a pending-count badge with a
 * manual "Sync Now" action, and a conflict warning when server and local edits
 * diverge. It renders nothing when everything is synced and online.
 */
export function SyncStatusBanner() {
  const { isOnline, isSyncing, pendingCount, conflictCount, syncNow } = useOnlineSync();

  const hasPending = pendingCount > 0;
  const hasConflicts = conflictCount > 0;
  if (isOnline && !hasPending && !hasConflicts) return null;

  return (
    <div className="border-b border-blue-200 bg-blue-50 px-4 py-2 text-sm text-blue-800">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3">
          {!isOnline && <span>Offline — alterações ficam na fila.</span>}
          {hasPending && (
            <span className="inline-flex items-center gap-1">
              <span
                aria-label="registros pendentes"
                className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-blue-600 px-1.5 text-xs font-semibold text-white"
              >
                {pendingCount}
              </span>
              {pendingCount === 1 ? 'registro pendente' : 'registros pendentes'}
            </span>
          )}
          {hasConflicts && (
            <span className="inline-flex items-center gap-1 font-medium text-amber-700">
              <span
                aria-label="conflitos"
                className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-amber-500 px-1.5 text-xs font-semibold text-white"
              >
                {conflictCount}
              </span>
              {conflictCount === 1 ? 'conflito' : 'conflitos'} de sincronização
            </span>
          )}
        </div>
        {hasPending && (
          <button
            type="button"
            onClick={() => void syncNow()}
            disabled={!isOnline || isSyncing}
            className="rounded-md bg-blue-600 px-3 py-1 text-xs font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSyncing ? 'Sincronizando…' : 'Sincronizar agora'}
          </button>
        )}
      </div>
    </div>
  );
}
