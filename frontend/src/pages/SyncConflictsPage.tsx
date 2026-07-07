import { useEffect, useState } from 'react';
import { useSyncConflicts } from '../hooks/useSyncConflicts';
import { ConflictList } from '../components/ui/ConflictList';
import { ConflictDiff } from '../components/ui/ConflictDiff';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import type { ConflictSnapshot, SyncEntityType, SyncQueueItem } from '../offline/db';

const ENTITY_LABELS: Record<SyncEntityType, string> = {
  person: 'Pessoa',
  triage: 'Triagem',
  attendance: 'Atendimento',
};

/** null marks a conflict whose snapshot was looked up but not found. */
type SnapshotMap = Record<string, ConflictSnapshot | null>;

interface ConflictDiffCardProps {
  item: SyncQueueItem;
  snapshot: ConflictSnapshot;
  onKeepLocal: (queueId: number) => void;
  onKeepServer: (queueId: number) => void;
  onMerge: (queueId: number, merged: Record<string, unknown>) => void;
}

function ConflictDiffCard({
  item,
  snapshot,
  onKeepLocal,
  onKeepServer,
  onMerge,
}: ConflictDiffCardProps) {
  const queueId = item.id as number;
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4">
      <header className="mb-3">
        <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
          {ENTITY_LABELS[item.entityType]}
        </p>
      </header>
      <ConflictDiff
        localData={item.data}
        snapshot={snapshot}
        onKeepLocal={() => onKeepLocal(queueId)}
        onKeepServer={() => onKeepServer(queueId)}
        onMerge={(merged) => onMerge(queueId, merged)}
      />
    </section>
  );
}

/**
 * useSnapshots loads the captured server version for every conflict into a map
 * keyed by entityId. A conflict with no snapshot maps to null so the page can
 * fall back to the simple resubmit/discard row. The map reloads whenever the
 * conflict set changes.
 */
function useSnapshots(
  conflicts: SyncQueueItem[],
  snapshotFor: (entityId: string) => Promise<ConflictSnapshot | undefined>,
): SnapshotMap {
  const [snapshots, setSnapshots] = useState<SnapshotMap>({});
  const key = conflicts.map((c) => c.entityId).join('|');

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const next: SnapshotMap = {};
      for (const item of conflicts) {
        next[item.entityId] = (await snapshotFor(item.entityId)) ?? null;
      }
      if (!cancelled) setSnapshots(next);
    })();
    return () => {
      cancelled = true;
    };
    // `key` captures the identity of the conflict set; snapshotFor is stable.
  }, [key, snapshotFor]); // eslint-disable-line react-hooks/exhaustive-deps

  return snapshots;
}

/**
 * SyncConflictsPage is the operator surface for records the server rejected as
 * conflicts during offline sync. When the server's version was captured, the
 * page shows a field-level diff (ConflictDiff) with keep-local / keep-server /
 * merge actions (S12.2). Conflicts without a captured snapshot fall back to the
 * simple resubmit / discard row so nothing regresses.
 */
export function SyncConflictsPage() {
  const {
    conflicts,
    isLoading,
    discard,
    resubmit,
    keepLocal,
    keepServer,
    merge,
    snapshotFor,
  } = useSyncConflicts();
  const snapshots = useSnapshots(conflicts, snapshotFor);

  if (isLoading) return <LoadingScreen />;

  const withSnapshot = conflicts.filter(
    (item): item is SyncQueueItem => snapshots[item.entityId] != null,
  );
  const withoutSnapshot = conflicts.filter((item) => snapshots[item.entityId] == null);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-semibold text-gray-900">Conflitos de sincronização</h1>
        <p className="text-sm text-gray-500">
          Registros que o servidor rejeitou. Revise as diferenças e escolha manter
          a versão local, a do servidor ou mesclar os campos.
        </p>
      </div>

      {conflicts.length === 0 && (
        <EmptyState
          title="Nenhum conflito"
          description="Todos os registros foram sincronizados sem conflitos."
        />
      )}

      {withSnapshot.length > 0 && (
        <div className="space-y-4">
          {withSnapshot.map((item) => (
            <ConflictDiffCard
              key={item.id}
              item={item}
              snapshot={snapshots[item.entityId] as ConflictSnapshot}
              onKeepLocal={(id) => void keepLocal(id)}
              onKeepServer={(id) => void keepServer(id)}
              onMerge={(id, merged) => void merge(id, merged)}
            />
          ))}
        </div>
      )}

      {withoutSnapshot.length > 0 && (
        <ConflictList
          conflicts={withoutSnapshot}
          onResubmit={(id) => void resubmit(id)}
          onDiscard={(id) => void discard(id)}
        />
      )}
    </div>
  );
}
