import { Link, useNavigate } from 'react-router-dom';
import { useDonations } from '../hooks/useDonations';
import { useAuth } from '../hooks/useAuth';
import { EmptyState } from '../components/ui/EmptyState';
import { Pagination } from '../components/ui/Pagination';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { DONATION_TYPE_LABELS, donationTypeLabel, formatCurrency } from '../utils/donationLabels';
import type { DonationListItem } from '../types';

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function DonationTable({
  donations,
  canManage,
}: {
  donations: DonationListItem[];
  canManage: boolean;
}) {
  const navigate = useNavigate();
  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Data</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Tipo</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Valor</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Doador</th>
            <th className="px-4 py-2 text-left font-medium text-gray-600">Campanha</th>
            {canManage && <th className="px-4 py-2 text-right font-medium text-gray-600">Ações</th>}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {donations.map((d) => (
            <tr
              key={d.id}
              className="cursor-pointer hover:bg-gray-50"
              onClick={() => navigate(`/donations/${d.id}`)}
            >
              <td className="px-4 py-3 text-gray-600">{formatDate(d.donation_date)}</td>
              <td className="px-4 py-3 text-gray-700">{donationTypeLabel(d.donation_type)}</td>
              <td className="px-4 py-3 text-gray-900">{formatCurrency(d.amount, d.currency)}</td>
              <td className="px-4 py-3 text-gray-700">{d.donor_name || 'Anônimo'}</td>
              <td className="px-4 py-3 text-gray-600">{d.campaign_name || '—'}</td>
              {canManage && (
                <td className="px-4 py-3 text-right">
                  <Link
                    to={`/donations/${d.id}/edit`}
                    className="text-sm font-medium text-blue-600 hover:text-blue-700"
                    onClick={(e) => e.stopPropagation()}
                  >
                    Editar
                  </Link>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TypeFilter({ onChange }: { onChange: (type: string | undefined) => void }) {
  return (
    <div className="max-w-xs">
      <label htmlFor="donation-type-filter" className="block text-sm font-medium text-gray-700">
        Tipo
      </label>
      <select
        id="donation-type-filter"
        className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm"
        defaultValue=""
        onChange={(e) => onChange(e.target.value || undefined)}
      >
        <option value="">Todos</option>
        {Object.entries(DONATION_TYPE_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
    </div>
  );
}

export function DonationListPage() {
  const { hasMinRole } = useAuth();
  const canManage = hasMinRole('SECRETARY');
  const { donations, pagination, isLoading, error, goToPage, filterByType } = useDonations();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Doações</h1>
        {canManage && (
          <Link
            to="/donations/new"
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Nova doação
          </Link>
        )}
      </div>

      <TypeFilter onChange={filterByType} />

      {error && <p className="text-center text-sm text-red-600">{error}</p>}

      {isLoading ? (
        <LoadingScreen />
      ) : donations.length === 0 ? (
        <EmptyState
          title="Nenhuma doação encontrada"
          description="Doações registram contribuições financeiras, bens e serviços recebidos."
        />
      ) : (
        <>
          <DonationTable donations={donations} canManage={canManage} />
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
