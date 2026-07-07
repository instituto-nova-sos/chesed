import { Link, useNavigate } from 'react-router-dom';
import { useAttendances } from '../hooks/useAttendances';
import { useCampusTimezone } from '../hooks/useCampusTimezone';
import { formatDateTime } from '../utils/formatDateTime';
import { EmptyState } from '../components/ui/EmptyState';
import { Pagination } from '../components/ui/Pagination';
import { LoadingScreen } from '../components/ui/LoadingScreen';
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

const STATUS_FILTERS: { value: AttendanceStatus | ''; label: string }[] = [
  { value: '', label: 'Todos' },
  { value: 'SCHEDULED', label: 'Agendado' },
  { value: 'IN_PROGRESS', label: 'Em Andamento' },
  { value: 'COMPLETED', label: 'Concluído' },
  { value: 'CANCELLED', label: 'Cancelado' },
];

type AttendanceRow = ReturnType<typeof useAttendances>['attendances'][number];

interface StatusFilterBarProps {
  activeStatus: AttendanceStatus | undefined;
  onFilter: (status: AttendanceStatus | undefined) => void;
}

function StatusFilterBar({ activeStatus, onFilter }: StatusFilterBarProps) {
  return (
    <div className="flex flex-wrap gap-1">
      {STATUS_FILTERS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onFilter(opt.value === '' ? undefined : opt.value)}
          className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
            (activeStatus ?? '') === opt.value
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

interface AttendanceTableRowProps {
  attendance: AttendanceRow;
  timeZone: string | undefined;
  onOpen: (id: string) => void;
}

function AttendanceTableRow({ attendance: a, timeZone, onOpen }: AttendanceTableRowProps) {
  return (
    <tr className="cursor-pointer hover:bg-gray-50" onClick={() => onOpen(a.id)}>
      <td className="px-4 py-3">
        <Link
          to={`/persons/${a.person_id}`}
          className="text-blue-600 hover:underline"
          onClick={(e) => e.stopPropagation()}
        >
          {a.person_name}
        </Link>
      </td>
      <td className="px-4 py-3 text-gray-700">{a.service_type}</td>
      <td className="px-4 py-3">
        <span
          className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[a.status]}`}
        >
          {STATUS_LABELS[a.status]}
        </span>
      </td>
      <td className="px-4 py-3 text-gray-600">
        {formatDateTime(a.attendance_date, timeZone)}
      </td>
    </tr>
  );
}

interface AttendanceTableProps {
  attendances: AttendanceRow[];
  timeZone: string | undefined;
  onOpen: (id: string) => void;
}

function AttendanceTable({ attendances, timeZone, onOpen }: AttendanceTableProps) {
  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Pessoa</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Serviço</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Status</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Data</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {attendances.map((a) => (
            <AttendanceTableRow key={a.id} attendance={a} timeZone={timeZone} onOpen={onOpen} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function AttendanceListPage() {
  const navigate = useNavigate();
  const timeZone = useCampusTimezone();
  const { attendances, pagination, isLoading, error, goToPage, filterByStatus, params } =
    useAttendances();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Atendimentos</h1>
      </div>

      <StatusFilterBar activeStatus={params.status} onFilter={filterByStatus} />

      {error && <p className="text-center text-sm text-red-600">{error}</p>}

      {isLoading ? (
        <LoadingScreen />
      ) : attendances.length === 0 ? (
        <EmptyState
          title="Nenhum atendimento encontrado"
          description="Atendimentos são criados a partir de uma triagem."
        />
      ) : (
        <>
          <AttendanceTable
            attendances={attendances}
            timeZone={timeZone}
            onOpen={(attendanceId) => navigate(`/attendances/${attendanceId}`)}
          />
          <Pagination
            page={pagination.page}
            totalPages={pagination.total_pages}
            onPageChange={goToPage}
          />
        </>
      )}
    </div>
  );
}
