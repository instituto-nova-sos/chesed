import { useSyncConflicts } from '../hooks/useSyncConflicts';
import { ConflictList } from '../components/ui/ConflictList';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';

/**
 * SyncConflictsPage is the operator surface for records the server rejected as
 * conflicts during offline sync. Each conflicted record is preserved (never
 * auto-dropped); the operator either resubmits it (last-write-wins retry) or
 * discards it after review.
 */
export function SyncConflictsPage() {
  const { conflicts, isLoading, discard, resubmit } = useSyncConflicts();

  if (isLoading) return <LoadingScreen />;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-semibold text-gray-900">Conflitos de sincronização</h1>
        <p className="text-sm text-gray-500">
          Registros que o servidor rejeitou. Reenvie para tentar novamente ou
          descarte após revisar.
        </p>
      </div>

      {conflicts.length === 0 ? (
        <EmptyState
          title="Nenhum conflito"
          description="Todos os registros foram sincronizados sem conflitos."
        />
      ) : (
        <ConflictList
          conflicts={conflicts}
          onResubmit={(id) => void resubmit(id)}
          onDiscard={(id) => void discard(id)}
        />
      )}
    </div>
  );
}
