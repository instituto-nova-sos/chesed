import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useAttendance } from '../hooks/useAttendance';
import { useCampusTimezone } from '../hooks/useCampusTimezone';
import { formatDateTime } from '../utils/formatDateTime';
import { transitionAttendance, updateAttendanceNotes } from '../api/attendances';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import type { AttendanceStatus } from '../types';

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

export function AttendanceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const timeZone = useCampusTimezone();
  const { attendance, isLoading, error, refetch } = useAttendance(id);
  const [transitioning, setTransitioning] = useState<AttendanceStatus | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [observations, setObservations] = useState<string>('');
  const [recommendations, setRecommendations] = useState<string>('');
  const [notesDirty, setNotesDirty] = useState(false);
  const [savingNotes, setSavingNotes] = useState(false);

  if (isLoading) return <LoadingScreen />;
  if (error || !attendance) {
    return <Alert variant="error">{error ?? 'Atendimento não encontrado'}</Alert>;
  }

  if (!notesDirty) {
    if ((observations || '') !== (attendance.observations ?? '')) {
      setObservations(attendance.observations ?? '');
    }
    if ((recommendations || '') !== (attendance.recommendations ?? '')) {
      setRecommendations(attendance.recommendations ?? '');
    }
  }

  async function doTransition(to: AttendanceStatus) {
    if (!id) return;
    setTransitioning(to);
    setActionError(null);
    try {
      await transitionAttendance(id, { to });
      refetch();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Falha na transição');
    } finally {
      setTransitioning(null);
    }
  }

  async function saveNotes() {
    if (!id) return;
    setSavingNotes(true);
    setActionError(null);
    try {
      await updateAttendanceNotes(id, {
        observations: observations || undefined,
        recommendations: recommendations || undefined,
      });
      setNotesDirty(false);
      refetch();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Falha ao salvar notas');
    } finally {
      setSavingNotes(false);
    }
  }

  const nextStates = NEXT_STATES[attendance.status];

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold text-gray-900">Atendimento</h1>
        <span
          className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_COLORS[attendance.status]}`}
        >
          {STATUS_LABELS[attendance.status]}
        </span>
      </div>

      {actionError && <Alert variant="error">{actionError}</Alert>}

      <dl className="grid grid-cols-1 gap-4 rounded-lg border border-gray-200 bg-white p-4 sm:grid-cols-2">
        <div>
          <dt className="text-xs font-medium text-gray-500">Pessoa</dt>
          <dd className="mt-1 text-sm">
            <Link
              to={`/persons/${attendance.person_id}`}
              className="text-blue-600 hover:underline"
            >
              Ver pessoa
            </Link>
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium text-gray-500">Data</dt>
          <dd className="mt-1 text-sm text-gray-900">
            {formatDateTime(attendance.attendance_date, timeZone)}
          </dd>
        </div>
        {attendance.triage_id && (
          <div>
            <dt className="text-xs font-medium text-gray-500">Triagem</dt>
            <dd className="mt-1 text-sm">
              <Link
                to={`/triages/${attendance.triage_id}`}
                className="text-blue-600 hover:underline"
              >
                Ver triagem
              </Link>
            </dd>
          </div>
        )}
      </dl>

      {nextStates.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-3 text-sm font-medium text-gray-900">Ações</h2>
          <div className="flex flex-wrap gap-2">
            {nextStates.map((next) => (
              <Button
                key={next}
                variant={next === 'CANCELLED' ? 'secondary' : 'primary'}
                onClick={() => doTransition(next)}
                disabled={transitioning !== null}
              >
                {transitioning === next ? '...' : ACTION_LABEL[next]}
              </Button>
            ))}
          </div>
        </div>
      )}

      <div className="rounded-lg border border-gray-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-medium text-gray-900">Anotações</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-xs font-medium text-gray-700">Observações</label>
            <textarea
              value={observations}
              onChange={(e) => {
                setObservations(e.target.value);
                setNotesDirty(true);
              }}
              rows={3}
              className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Recomendações</label>
            <textarea
              value={recommendations}
              onChange={(e) => {
                setRecommendations(e.target.value);
                setNotesDirty(true);
              }}
              rows={3}
              className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <Button onClick={saveNotes} disabled={!notesDirty || savingNotes}>
            {savingNotes ? 'Salvando...' : 'Salvar Anotações'}
          </Button>
        </div>
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-medium text-gray-900">Histórico de Status</h2>
        {attendance.transitions.length === 0 ? (
          <p className="text-sm text-gray-500">Sem transições registradas.</p>
        ) : (
          <ol className="space-y-2 text-sm">
            {attendance.transitions.map((t) => (
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
    </div>
  );
}
