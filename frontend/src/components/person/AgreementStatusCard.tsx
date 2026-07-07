import { useEffect, useState } from 'react';
import { useAuth } from '../../hooks/useAuth';
import { getPersonAgreement, downloadAgreementDocument } from '../../api/persons';
import { Button } from '../ui/Button';
import { AgreementUploadModal } from './AgreementUploadModal';
import type { VolunteerAgreement } from '../../types';

interface AgreementStatusCardProps {
  personId: string;
  hasVolunteerRole: boolean;
}

type StatusBadge = { label: string; classes: string };
const FALLBACK_STATUS: StatusBadge = {
  label: 'Pendente',
  classes: 'bg-yellow-100 text-yellow-800',
};
const statusLabels: Record<string, StatusBadge> = {
  ACCEPTED: { label: 'Aceito', classes: 'bg-green-100 text-green-800' },
  PENDING: FALLBACK_STATUS,
  REJECTED: { label: 'Recusado', classes: 'bg-red-100 text-red-800' },
};

const methodLabels: Record<string, string> = {
  DIGITAL: 'Assinatura Digital',
  MANUAL_UPLOAD: 'Documento Enviado',
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR');
}

function AgreementDetails({ agreement }: { agreement: VolunteerAgreement }) {
  return (
    <div className="space-y-1 text-sm text-gray-600">
      {agreement.signature_method && (
        <p>Metodo: {methodLabels[agreement.signature_method] ?? agreement.signature_method}</p>
      )}
      {agreement.accepted_at && <p>Aceito em: {formatDate(agreement.accepted_at)}</p>}
      {agreement.rejected_at && <p>Recusado em: {formatDate(agreement.rejected_at)}</p>}
      {agreement.rejection_reason && <p>Motivo: {agreement.rejection_reason}</p>}
    </div>
  );
}

interface AgreementActionsProps {
  personId: string;
  status: string;
  hasDocument: boolean;
  onUpload: () => void;
}

function AgreementActions({ personId, status, hasDocument, onUpload }: AgreementActionsProps) {
  const canUpload = status === 'PENDING' || status === 'REJECTED';
  return (
    <div className="mt-3 flex gap-2">
      {canUpload && (
        <Button size="sm" variant="secondary" onClick={onUpload}>
          Enviar Termo Assinado
        </Button>
      )}
      {hasDocument && (
        <a
          href={downloadAgreementDocument(personId)}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          Baixar Documento
        </a>
      )}
    </div>
  );
}

export function AgreementStatusCard({ personId, hasVolunteerRole }: AgreementStatusCardProps) {
  const { hasMinRole } = useAuth();
  const [agreements, setAgreements] = useState<VolunteerAgreement[]>([]);
  const [loading, setLoading] = useState(true);
  const [showUpload, setShowUpload] = useState(false);
  const canManage = hasMinRole('COORDINATOR');

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!hasVolunteerRole || !canManage) {
        setLoading(false);
        return;
      }
      try {
        const { agreements: data } = await getPersonAgreement(personId);
        if (!cancelled) setAgreements(data);
      } catch {
        if (!cancelled) setAgreements([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [personId, hasVolunteerRole, canManage]);

  if (!hasVolunteerRole) return null;

  if (loading) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <div className="h-6 w-32 animate-pulse rounded bg-gray-200" />
      </div>
    );
  }

  const latestAgreement = agreements[0];
  const status = latestAgreement?.status ?? 'PENDING';
  const statusInfo = statusLabels[status] ?? FALLBACK_STATUS;

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-900">
          Termo de Voluntario
        </h3>
        <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${statusInfo.classes}`}>
          {statusInfo.label}
        </span>
      </div>

      {latestAgreement && <AgreementDetails agreement={latestAgreement} />}

      {canManage && (
        <AgreementActions
          personId={personId}
          status={status}
          hasDocument={Boolean(latestAgreement?.document_path)}
          onUpload={() => setShowUpload(true)}
        />
      )}

      {showUpload && (
        <AgreementUploadModal
          personId={personId}
          onClose={() => setShowUpload(false)}
          onUploaded={(agreement) => {
            setAgreements([agreement, ...agreements]);
            setShowUpload(false);
          }}
        />
      )}
    </div>
  );
}
