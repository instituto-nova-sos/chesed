import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useAttendance } from '../hooks/useAttendance';
import { useCampusTimezone } from '../hooks/useCampusTimezone';
import { formatDateTime } from '../utils/formatDateTime';
import { transitionAttendance, updateAttendanceNotes } from '../api/attendances';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Alert } from '../components/ui/Alert';
import {
  StatusBadge,
  TransitionPanel,
  NotesPanel,
  HistoryPanel,
} from './AttendanceDetailPanels';
import type { AttendanceStatus, AttendanceDetail } from '../types';

function AttendanceSummary({
  attendance,
  timeZone,
}: {
  attendance: AttendanceDetail;
  timeZone: string | undefined;
}) {
  return (
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
  );
}

interface NotesValues {
  observations: string;
  recommendations: string;
}

interface NotesDraft extends NotesValues {
  notesDirty: boolean;
  savingNotes: boolean;
  setObservations: (value: string) => void;
  setRecommendations: (value: string) => void;
  saveNotes: () => Promise<void>;
}

function serverNotes(attendance: AttendanceDetail | null): NotesValues {
  return {
    observations: attendance?.observations ?? '',
    recommendations: attendance?.recommendations ?? '',
  };
}

function useNotesDraft(
  id: string | undefined,
  attendance: AttendanceDetail | null,
  refetch: () => void,
  onError: (message: string | null) => void,
): NotesDraft {
  const [draft, setDraft] = useState<NotesValues>(() => serverNotes(attendance));
  const [notesDirty, setNotesDirty] = useState(false);
  const [savingNotes, setSavingNotes] = useState(false);

  // Sync draft from server value during render while the user hasn't edited it.
  // Kept in render (not an effect) to avoid a flash of stale text on refetch.
  const fromServer = serverNotes(attendance);
  const isStale =
    draft.observations !== fromServer.observations ||
    draft.recommendations !== fromServer.recommendations;
  if (!notesDirty && isStale) {
    setDraft(fromServer);
  }

  const edit = (patch: Partial<NotesValues>) => {
    setDraft((prev) => ({ ...prev, ...patch }));
    setNotesDirty(true);
  };

  async function saveNotes() {
    if (!id) return;
    setSavingNotes(true);
    onError(null);
    try {
      await updateAttendanceNotes(id, {
        observations: draft.observations || undefined,
        recommendations: draft.recommendations || undefined,
      });
      setNotesDirty(false);
      refetch();
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Falha ao salvar notas');
    } finally {
      setSavingNotes(false);
    }
  }

  return {
    ...draft,
    notesDirty,
    savingNotes,
    setObservations: (value) => edit({ observations: value }),
    setRecommendations: (value) => edit({ recommendations: value }),
    saveNotes,
  };
}

export function AttendanceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const timeZone = useCampusTimezone();
  const { attendance, isLoading, error, refetch } = useAttendance(id);
  const [transitioning, setTransitioning] = useState<AttendanceStatus | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const notes = useNotesDraft(id, attendance, refetch, setActionError);

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

  if (isLoading) return <LoadingScreen />;
  if (error || !attendance) {
    return <Alert variant="error">{error ?? 'Atendimento não encontrado'}</Alert>;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold text-gray-900">Atendimento</h1>
        <StatusBadge status={attendance.status} />
      </div>

      {actionError && <Alert variant="error">{actionError}</Alert>}

      <AttendanceSummary attendance={attendance} timeZone={timeZone} />

      <TransitionPanel
        status={attendance.status}
        transitioning={transitioning}
        onTransition={doTransition}
      />

      <NotesPanel
        observations={notes.observations}
        recommendations={notes.recommendations}
        onObservationsChange={notes.setObservations}
        onRecommendationsChange={notes.setRecommendations}
        onSave={notes.saveNotes}
        notesDirty={notes.notesDirty}
        savingNotes={notes.savingNotes}
      />

      <HistoryPanel transitions={attendance.transitions} timeZone={timeZone} />
    </div>
  );
}
