import type { SyncQueueItem, SyncEntityType } from '../../offline/db';
import { Button } from './Button';

const ENTITY_LABELS: Record<SyncEntityType, string> = {
  person: 'Pessoa',
  triage: 'Triagem',
  attendance: 'Atendimento',
};

function recordSummary(item: SyncQueueItem): string {
  const data = item.data as { full_name?: string; main_complaint?: string };
  return data.full_name ?? data.main_complaint ?? item.entityId;
}

interface ConflictRowProps {
  item: SyncQueueItem;
  onResubmit: (queueId: number) => void;
  onDiscard: (queueId: number) => void;
}

function ConflictRow({ item, onResubmit, onDiscard }: ConflictRowProps) {
  const queueId = item.id as number;
  return (
    <li className="flex flex-col gap-2 border-b border-gray-100 py-3 last:border-0 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="truncate font-medium text-gray-900">{recordSummary(item)}</p>
        <p className="text-xs text-gray-500">
          {ENTITY_LABELS[item.entityType]}
          {item.lastError ? ` — ${item.lastError}` : ''}
        </p>
      </div>
      <div className="flex shrink-0 gap-2">
        <Button variant="secondary" size="sm" onClick={() => onResubmit(queueId)}>
          Reenviar
        </Button>
        <Button variant="danger" size="sm" onClick={() => onDiscard(queueId)}>
          Descartar
        </Button>
      </div>
    </li>
  );
}

interface ConflictListProps {
  conflicts: SyncQueueItem[];
  onResubmit: (queueId: number) => void;
  onDiscard: (queueId: number) => void;
}

export function ConflictList({ conflicts, onResubmit, onDiscard }: ConflictListProps) {
  return (
    <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white px-4">
      {conflicts.map((item) => (
        <ConflictRow
          key={item.id}
          item={item}
          onResubmit={onResubmit}
          onDiscard={onDiscard}
        />
      ))}
    </ul>
  );
}
