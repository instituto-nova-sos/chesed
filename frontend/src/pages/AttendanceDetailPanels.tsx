import { formatDateTime } from '../utils/formatDateTime';
import { Button } from '../components/ui/Button';
import type { AttendanceStatus, AttendanceDetail } from '../types';

const STATUS_LABELS: Record<AttendanceStatus, string> = {
  SCHEDULED: 'Agendado',
  IN_PROGRESS: 'Em Andamento',
  COMPLETED: 'Concluído',
  CANCELLED: 'Cancelado',
};

const STATUS_COLORS: Record<AttendanceStatus, string> = {
  SCHEDULED: 'bg-blue-100 text-blue-800',
  IN_PROGRESS: 'bg-amber-100 text-amber-800',
  COMPLETED: 'bg-green-100 text-green-800',
  CANCELLED: 'bg-gray-100 text-gray-700',
};

// Allowed UI transitions per Phase 1 state machine.
const NEXT_STATES: Record<AttendanceStatus, AttendanceStatus[]> = {
  SCHEDULED: ['IN_PROGRESS', 'CANCELLED'],
  IN_PROGRESS: ['COMPLETED', 'CANCELLED'],
  COMPLETED: [],
  CANCELLED: [],
};

const ACTION_LABEL: Record<AttendanceStatus, string> = {
  SCHEDULED: 'Reagendar',
  IN_PROGRESS: 'Iniciar',
  COMPLETED: 'Concluir',
  CANCELLED: 'Cancelar',
};

const NOTES_TEXTAREA_CLASS =
  'mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500';

export function StatusBadge({ status }: { status: AttendanceStatus }) {
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_COLORS[status]}`}
    >
      {STATUS_LABELS[status]}
    </span>
  );
}

interface TransitionPanelProps {
  status: AttendanceStatus;
  transitioning: AttendanceStatus | null;
  onTransition: (to: AttendanceStatus) => void;
}

export function TransitionPanel({ status, transitioning, onTransition }: TransitionPanelProps) {
  const nextStates = NEXT_STATES[status];
  if (nextStates.length === 0) return null;
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <h2 className="mb-3 text-sm font-medium text-gray-900">Ações</h2>
      <div className="flex flex-wrap gap-2">
        {nextStates.map((next) => (
          <Button
            key={next}
            variant={next === 'CANCELLED' ? 'secondary' : 'primary'}
            onClick={() => onTransition(next)}
            disabled={transitioning !== null}
          >
            {transitioning === next ? '...' : ACTION_LABEL[next]}
          </Button>
        ))}
      </div>
    </div>
  );
}

interface NotesPanelProps {
  observations: string;
  recommendations: string;
  onObservationsChange: (value: string) => void;
  onRecommendationsChange: (value: string) => void;
  onSave: () => void;
  notesDirty: boolean;
  savingNotes: boolean;
}

export function NotesPanel({
  observations,
  recommendations,
  onObservationsChange,
  onRecommendationsChange,
  onSave,
  notesDirty,
  savingNotes,
}: NotesPanelProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <h2 className="mb-3 text-sm font-medium text-gray-900">Anotações</h2>
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-gray-700">Observações</label>
          <textarea
            value={observations}
            onChange={(e) => onObservationsChange(e.target.value)}
            rows={3}
            className={NOTES_TEXTAREA_CLASS}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-700">Recomendações</label>
          <textarea
            value={recommendations}
            onChange={(e) => onRecommendationsChange(e.target.value)}
            rows={3}
            className={NOTES_TEXTAREA_CLASS}
          />
        </div>
        <Button onClick={onSave} disabled={!notesDirty || savingNotes}>
          {savingNotes ? 'Salvando...' : 'Salvar Anotações'}
        </Button>
      </div>
    </div>
  );
}

interface HistoryPanelProps {
  transitions: AttendanceDetail['transitions'];
  timeZone: string | undefined;
}

export function HistoryPanel({ transitions, timeZone }: HistoryPanelProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <h2 className="mb-3 text-sm font-medium text-gray-900">Histórico de Status</h2>
      {transitions.length === 0 ? (
        <p className="text-sm text-gray-500">Sem transições registradas.</p>
      ) : (
        <ol className="space-y-2 text-sm">
          {transitions.map((t) => (
            <li key={t.id} className="flex items-center gap-2">
              <span className="text-gray-500">{formatDateTime(t.transitioned_at, timeZone)}</span>
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[t.from_status]}`}
              >
                {STATUS_LABELS[t.from_status]}
              </span>
              <span className="text-gray-400">→</span>
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[t.to_status]}`}
              >
                {STATUS_LABELS[t.to_status]}
              </span>
              {t.reason && <span className="text-gray-600">— {t.reason}</span>}
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}
