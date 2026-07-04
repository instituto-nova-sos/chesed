import type { Consent } from '../../types/consent';
import { consentStatusLabel, consentTypeLabel } from '../../utils/consentLabels';

interface ConsentHistoryListProps {
  consents: Consent[];
  canRevoke: boolean;
  onRevoke: (consent: Consent) => void;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function StatusBadge({ isActive }: { isActive: boolean }) {
  const colors = isActive ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800';
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${colors}`}>
      {consentStatusLabel(isActive)}
    </span>
  );
}

export function ConsentHistoryList({ consents, canRevoke, onRevoke }: ConsentHistoryListProps) {
  if (consents.length === 0) {
    return <p className="text-sm text-gray-500">Nenhum consentimento registrado.</p>;
  }

  return (
    <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
      {consents.map((consent) => (
        <li key={consent.id} className="space-y-1 px-4 py-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium text-gray-900">
                {consentTypeLabel(consent.consent_type)}
              </span>
              <StatusBadge isActive={consent.is_active} />
            </div>
            {canRevoke && consent.is_active && (
              <button
                type="button"
                onClick={() => onRevoke(consent)}
                className="text-sm font-medium text-red-600 hover:text-red-800"
              >
                Revogar
              </button>
            )}
          </div>
          <p className="text-xs text-gray-500">
            Versão {consent.consent_version} · Concedido em {formatDate(consent.granted_at)}
          </p>
          {consent.revoked_reason && (
            <p className="text-xs text-red-700">Motivo: {consent.revoked_reason}</p>
          )}
        </li>
      ))}
    </ul>
  );
}
