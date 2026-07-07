import { useMemo, useState } from 'react';
import type { ConflictSnapshot } from '../../offline/db';
import { Button } from './Button';

interface ConflictDiffProps {
  localData: Record<string, unknown>;
  snapshot: ConflictSnapshot;
  onKeepLocal: () => void;
  onKeepServer: () => void;
  onMerge: (merged: Record<string, unknown>) => void;
}

type FieldSource = 'local' | 'server';

/** unionKeys returns the local keys first, then any server-only keys, once each. */
function unionKeys(
  localData: Record<string, unknown>,
  serverData: Record<string, unknown>,
): string[] {
  const keys = Object.keys(localData);
  for (const key of Object.keys(serverData)) {
    if (!keys.includes(key)) keys.push(key);
  }
  return keys;
}

/** formatValue renders a cell value: primitives as text, objects as compact JSON. */
function formatValue(value: unknown): string {
  if (value === undefined || value === null) return '—';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

function defaultSelection(keys: string[], conflictFields: string[]): Record<string, FieldSource> {
  const selection: Record<string, FieldSource> = {};
  for (const key of keys) {
    selection[key] = conflictFields.includes(key) ? 'server' : 'local';
  }
  return selection;
}

interface FieldRowProps {
  fieldKey: string;
  localValue: unknown;
  serverValue: unknown;
  isConflict: boolean;
  selected: FieldSource;
  onSelect: (source: FieldSource) => void;
}

function FieldRow({
  fieldKey,
  localValue,
  serverValue,
  isConflict,
  selected,
  onSelect,
}: FieldRowProps) {
  const rowClass = isConflict
    ? 'bg-amber-50 ring-1 ring-amber-300'
    : 'bg-white';
  return (
    <div
      data-testid={isConflict ? `conflict-field-${fieldKey}` : undefined}
      className={`flex flex-col gap-2 rounded-md p-3 sm:grid sm:grid-cols-[8rem_1fr_1fr] sm:items-center sm:gap-4 ${rowClass}`}
    >
      <span className="break-all font-mono text-xs font-medium text-gray-700">
        {fieldKey}
      </span>
      <div className="min-w-0 break-words text-sm text-gray-900">
        {formatValue(localValue)}
      </div>
      <div className="min-w-0 break-words text-sm text-gray-900">
        {formatValue(serverValue)}
      </div>
      {isConflict && (
        <div className="flex gap-2 sm:col-span-3">
          <Button
            variant={selected === 'local' ? 'primary' : 'secondary'}
            size="sm"
            aria-pressed={selected === 'local'}
            onClick={() => onSelect('local')}
          >
            {`${fieldKey}: usar local`}
          </Button>
          <Button
            variant={selected === 'server' ? 'primary' : 'secondary'}
            size="sm"
            aria-pressed={selected === 'server'}
            onClick={() => onSelect('server')}
          >
            {`${fieldKey}: usar servidor`}
          </Button>
        </div>
      )}
    </div>
  );
}

/**
 * ConflictDiff presents a field-by-field comparison between the operator's local
 * record and the server's captured version, and offers three resolutions:
 * keep the whole local record, keep the whole server record, or apply a
 * per-field merge. Conflicting fields (from snapshot.conflictFields) are
 * highlighted and default to the server value (last-write-wins); non-conflicting
 * fields default to local. The merged payload is built from the current
 * selection when "Aplicar mesclagem" is pressed.
 */
export function ConflictDiff({
  localData,
  snapshot,
  onKeepLocal,
  onKeepServer,
  onMerge,
}: ConflictDiffProps) {
  const { serverData } = snapshot;
  const conflictFields = snapshot.conflictFields ?? [];
  const keys = useMemo(() => unionKeys(localData, serverData), [localData, serverData]);
  const [selection, setSelection] = useState<Record<string, FieldSource>>(() =>
    defaultSelection(keys, conflictFields),
  );

  const buildMerged = (): Record<string, unknown> => {
    const merged: Record<string, unknown> = {};
    for (const key of keys) {
      merged[key] = selection[key] === 'local' ? localData[key] : serverData[key];
    }
    return merged;
  };

  return (
    <div className="space-y-3">
      <div className="hidden grid-cols-[8rem_1fr_1fr] gap-4 px-3 text-xs font-semibold uppercase tracking-wide text-gray-500 sm:grid">
        <span>Campo</span>
        <span>Local</span>
        <span>Servidor</span>
      </div>
      <div className="space-y-2">
        {keys.map((key) => (
          <FieldRow
            key={key}
            fieldKey={key}
            localValue={localData[key]}
            serverValue={serverData[key]}
            isConflict={conflictFields.includes(key)}
            selected={selection[key] ?? 'local'}
            onSelect={(source) => setSelection((prev) => ({ ...prev, [key]: source }))}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" size="sm" onClick={onKeepLocal}>
          Manter local
        </Button>
        <Button variant="secondary" size="sm" onClick={onKeepServer}>
          Manter servidor
        </Button>
        <Button variant="primary" size="sm" onClick={() => onMerge(buildMerged())}>
          Aplicar mesclagem
        </Button>
      </div>
    </div>
  );
}
