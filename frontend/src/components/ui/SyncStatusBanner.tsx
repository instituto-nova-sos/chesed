import { Link } from 'react-router-dom';
import { useOnlineSync } from '../../hooks/useOnlineSync';

interface SyncBannerState {
  isOnline: boolean;
  isSyncing: boolean;
  hasPending: boolean;
  hasConflicts: boolean;
}

function syncNowTitleFor(isOnline: boolean, isSyncing: boolean): string {
  if (!isOnline) {
    return 'Sem conexão — a sincronização ocorrerá automaticamente quando você voltar a ficar online.';
  }
  if (isSyncing) return 'Sincronização em andamento.';
  return 'Sincronizar os registros pendentes agora.';
}

// The banner hides only when there is nothing to communicate: online, idle, and
// fully synced. It stays mounted while syncing so an active drain is always
// visible, even if the pending count momentarily reads zero mid-flight.
function bannerIsHidden(s: SyncBannerState): boolean {
  return s.isOnline && !s.isSyncing && !s.hasPending && !s.hasConflicts;
}

function SyncingIndicator() {
  return (
    <span aria-live="polite" className="inline-flex items-center gap-1 text-blue-700">
      <span
        aria-hidden="true"
        className="h-3 w-3 animate-spin rounded-full border-2 border-blue-300 border-t-blue-600"
      />
      Sincronizando…
    </span>
  );
}

function PendingBadge({ count }: { count: number }) {
  return (
    <span className="inline-flex items-center gap-1">
      <span
        aria-label="registros pendentes"
        className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-blue-600 px-1.5 text-xs font-semibold text-white"
      >
        {count}
      </span>
      {count === 1 ? 'registro pendente' : 'registros pendentes'}
    </span>
  );
}

function ConflictBadge({ count }: { count: number }) {
  return (
    <Link
      to="/sync/conflicts"
      className="inline-flex items-center gap-1 font-medium text-amber-700 hover:underline"
    >
      <span
        aria-label="conflitos"
        className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-amber-500 px-1.5 text-xs font-semibold text-white"
      >
        {count}
      </span>
      {count === 1 ? 'conflito' : 'conflitos'} de sincronização
    </Link>
  );
}

/**
 * SyncStatusBanner surfaces the offline-sync state (S05.5): the offline notice,
 * a pending-count badge, a syncing indicator during an active drain, and a
 * conflict badge. The manual "Sync Now" action is disabled with an explanatory
 * title when offline or already syncing. It renders nothing only when online,
 * idle, and fully synced.
 */
export function SyncStatusBanner() {
  const { isOnline, isSyncing, pendingCount, conflictCount, syncNow } = useOnlineSync();

  const hasPending = pendingCount > 0;
  const hasConflicts = conflictCount > 0;
  if (bannerIsHidden({ isOnline, isSyncing, hasPending, hasConflicts })) return null;

  return (
    <div className="border-b border-blue-200 bg-blue-50 px-4 py-2 text-sm text-blue-800">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3">
          {!isOnline && <span>Offline — alterações ficam na fila.</span>}
          {isSyncing && <SyncingIndicator />}
          {hasPending && <PendingBadge count={pendingCount} />}
          {hasConflicts && <ConflictBadge count={conflictCount} />}
        </div>
        {(hasPending || hasConflicts) && (
          <button
            type="button"
            onClick={() => void syncNow()}
            disabled={!isOnline || isSyncing}
            title={syncNowTitleFor(isOnline, isSyncing)}
            className="rounded-md bg-blue-600 px-3 py-1 text-xs font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSyncing ? 'Sincronizando…' : 'Sincronizar agora'}
          </button>
        )}
      </div>
    </div>
  );
}
