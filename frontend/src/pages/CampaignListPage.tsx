import { Link, useNavigate } from 'react-router-dom';
import { useCampaigns } from '../hooks/useCampaigns';
import { useAuth } from '../hooks/useAuth';
import { EmptyState } from '../components/ui/EmptyState';
import { Pagination } from '../components/ui/Pagination';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Badge } from '../components/ui/Badge';
import {
  CAMPAIGN_STATUS_LABELS,
  campaignStatusLabel,
  campaignTypeLabel,
} from '../utils/campaignLabels';
import type { CampaignListItem } from '../types';

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function CampaignTable({ campaigns }: { campaigns: CampaignListItem[] }) {
  const navigate = useNavigate();
  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Nome</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Tipo</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Status</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Início</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Local</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {campaigns.map((c) => (
            <tr
              key={c.id}
              className="cursor-pointer hover:bg-gray-50"
              onClick={() => navigate(`/campaigns/${c.id}`)}
            >
              <td className="px-4 py-3 font-medium text-gray-900">{c.name}</td>
              <td className="px-4 py-3 text-gray-700">{campaignTypeLabel(c.campaign_type)}</td>
              <td className="px-4 py-3">
                <Badge label={campaignStatusLabel(c.status)} variant={c.status} />
              </td>
              <td className="px-4 py-3 text-gray-600">{formatDate(c.start_date)}</td>
              <td className="px-4 py-3 text-gray-600">{c.location_name ?? '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function StatusFilter({ onChange }: { onChange: (status: string | undefined) => void }) {
  return (
    <div className="max-w-xs">
      <label htmlFor="campaign-status-filter" className="block text-sm font-medium text-gray-700">
        Status
      </label>
      <select
        id="campaign-status-filter"
        className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm"
        defaultValue=""
        onChange={(e) => onChange(e.target.value || undefined)}
      >
        <option value="">Todos</option>
        {Object.entries(CAMPAIGN_STATUS_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
    </div>
  );
}

export function CampaignListPage() {
  const { hasMinRole } = useAuth();
  const { campaigns, pagination, isLoading, error, goToPage, filterByStatus } = useCampaigns();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Campanhas</h1>
        {hasMinRole('COORDINATOR') && (
          <Link
            to="/campaigns/new"
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Nova campanha
          </Link>
        )}
      </div>

      <StatusFilter onChange={filterByStatus} />

      {error && <p className="text-center text-sm text-red-600">{error}</p>}

      {isLoading ? (
        <LoadingScreen />
      ) : campaigns.length === 0 ? (
        <EmptyState
          title="Nenhuma campanha encontrada"
          description="Campanhas registram ações sociais e eventos do campus."
        />
      ) : (
        <>
          <CampaignTable campaigns={campaigns} />
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
