import { Link, useParams } from 'react-router-dom';
import { useDonation } from '../hooks/useDonation';
import { useAuth } from '../hooks/useAuth';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { donationTypeLabel, formatCurrency } from '../utils/donationLabels';
import type { DonationDetail } from '../types';

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function DonationInfo({ donation }: { donation: DonationDetail }) {
  const rows = [
    { label: 'Tipo', value: donationTypeLabel(donation.donation_type) },
    { label: 'Valor', value: formatCurrency(donation.amount, donation.currency) },
    { label: 'Item', value: donation.item_description ?? '—' },
    { label: 'Doador', value: donation.donor_name || 'Anônimo' },
    { label: 'Campanha', value: donation.campaign_name || '—' },
    { label: 'Data', value: formatDate(donation.donation_date) },
    { label: 'Observações', value: donation.notes ?? '—' },
  ];
  return (
    <dl className="grid grid-cols-1 gap-3 rounded-lg border border-gray-200 bg-white p-4 sm:grid-cols-2">
      {rows.map((row) => (
        <div key={row.label}>
          <dt className="text-xs font-medium uppercase text-gray-500">{row.label}</dt>
          <dd className="text-sm text-gray-900">{row.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function DonationDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { hasMinRole } = useAuth();
  const canManage = hasMinRole('SECRETARY');
  const { donation, isLoading, error } = useDonation(id);

  if (isLoading) return <LoadingScreen />;
  if (error || !donation) {
    return <Alert variant="error">{error ?? 'Doação não encontrada'}</Alert>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Doação</h1>
        {canManage && (
          <Link
            to={`/donations/${donation.id}/edit`}
            className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Editar
          </Link>
        )}
      </div>

      <DonationInfo donation={donation} />
    </div>
  );
}
