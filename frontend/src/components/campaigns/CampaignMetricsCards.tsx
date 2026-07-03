import type { CampaignMetrics } from '../../types';
import { campaignStatusLabel } from '../../utils/campaignLabels';

interface CampaignMetricsCardsProps {
  metrics: CampaignMetrics;
}

const STATUS_ORDER = ['COMPLETED', 'IN_PROGRESS', 'SCHEDULED', 'CANCELLED'];

export function CampaignMetricsCards({ metrics }: CampaignMetricsCardsProps) {
  const cards = [
    { label: 'Triagens', value: metrics.triage_count },
    { label: 'Atendimentos', value: metrics.attendance_total },
    { label: 'Equipe', value: metrics.team_size },
  ];

  const byStatus = STATUS_ORDER.filter((s) => metrics.attendances_by_status[s] !== undefined);

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {cards.map((card) => (
          <div key={card.label} className="rounded-lg border border-gray-200 bg-white p-4">
            <p className="text-sm text-gray-600">{card.label}</p>
            <p className="text-2xl font-semibold text-gray-900">{card.value}</p>
          </div>
        ))}
      </div>
      {byStatus.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <p className="mb-2 text-sm font-medium text-gray-700">Atendimentos por status</p>
          <ul className="space-y-1 text-sm text-gray-600">
            {byStatus.map((status) => (
              <li key={status} className="flex justify-between">
                <span>{attendanceStatusLabel(status)}</span>
                <span className="font-medium text-gray-900">
                  {metrics.attendances_by_status[status]}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

const ATTENDANCE_STATUS_LABELS: Record<string, string> = {
  SCHEDULED: 'Agendado',
  IN_PROGRESS: 'Em andamento',
  COMPLETED: 'Concluído',
  CANCELLED: 'Cancelado',
};

function attendanceStatusLabel(status: string): string {
  return ATTENDANCE_STATUS_LABELS[status] ?? campaignStatusLabel(status);
}
