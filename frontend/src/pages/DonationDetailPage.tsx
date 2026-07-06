import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useDonation } from '../hooks/useDonation';
import { useAuth } from '../hooks/useAuth';
import { downloadReceipt } from '../api/donations';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
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
  if (donation.receipt_number) {
    rows.push({ label: 'Recibo nº', value: donation.receipt_number });
    if (donation.receipt_issued_at) {
      rows.push({
        label: 'Recibo emitido em',
        value: formatDate(donation.receipt_issued_at),
      });
    }
  }
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
  const canDownloadReceipt = hasMinRole('COORDINATOR');
  const { donation, isLoading, error } = useDonation(id);
  const [receiptLoading, setReceiptLoading] = useState(false);
  const [receiptError, setReceiptError] = useState<string | null>(null);

  async function handleReceipt() {
    if (!donation) return;
    setReceiptLoading(true);
    setReceiptError(null);
    try {
      const { url } = await downloadReceipt(donation.id);
      window.open(url, '_blank', 'noopener,noreferrer');
    } catch (err) {
      setReceiptError(err instanceof Error ? err.message : 'Falha ao gerar recibo');
    } finally {
      setReceiptLoading(false);
    }
  }

  if (isLoading) return <LoadingScreen />;
  if (error || !donation) {
    return <Alert variant="error">{error ?? 'Doação não encontrada'}</Alert>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold text-gray-900">Doação</h1>
        <div className="flex items-center gap-3">
          {canDownloadReceipt && (
            <Button
              type="button"
              variant="secondary"
              onClick={handleReceipt}
              isLoading={receiptLoading}
            >
              Baixar recibo
            </Button>
          )}
          {canManage && (
            <Link
              to={`/donations/${donation.id}/edit`}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Editar
            </Link>
          )}
        </div>
      </div>

      {receiptError && <Alert variant="error">{receiptError}</Alert>}

      <DonationInfo donation={donation} />
    </div>
  );
}
