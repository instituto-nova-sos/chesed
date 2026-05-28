import { useEffect, useState } from 'react';
import { Link, useParams, useNavigate } from 'react-router-dom';
import { useTriage } from '../hooks/useTriage';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function TriageDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { triage, isLoading, error } = useTriage(id);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);

  useEffect(() => {
    listServiceTypes()
      .then(({ data }) => setServiceTypes(data))
      .catch(() => setServiceTypes([]));
  }, []);

  if (isLoading) return <LoadingScreen />;
  if (error || !triage) {
    return <Alert variant="error">{error ?? 'Triagem não encontrada'}</Alert>;
  }

  const stMap = new Map(serviceTypes.map((s) => [s.id, s.name]));
  const requested = (triage.requested_service_types ?? []).map(
    (id) => stMap.get(id) ?? id,
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Triagem</h1>
        <Button
          onClick={() =>
            navigate(`/attendances/new?person_id=${triage.person_id}&triage_id=${triage.id}`)
          }
        >
          Iniciar Atendimento
        </Button>
      </div>

      <dl className="grid grid-cols-1 gap-4 rounded-lg border border-gray-200 bg-white p-4 sm:grid-cols-2">
        <div>
          <dt className="text-xs font-medium text-gray-500">Pessoa</dt>
          <dd className="mt-1 text-sm">
            <Link
              to={`/persons/${triage.person_id}`}
              className="text-blue-600 hover:underline"
            >
              Ver pessoa
            </Link>
          </dd>
        </div>
        <div>
          <dt className="text-xs font-medium text-gray-500">Data</dt>
          <dd className="mt-1 text-sm text-gray-900">{formatDateTime(triage.triage_date)}</dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-xs font-medium text-gray-500">Queixa Principal</dt>
          <dd className="mt-1 whitespace-pre-wrap text-sm text-gray-900">
            {triage.main_complaint}
          </dd>
        </div>
        {triage.location && (
          <div className="sm:col-span-2">
            <dt className="text-xs font-medium text-gray-500">Local</dt>
            <dd className="mt-1 text-sm text-gray-900">{triage.location}</dd>
          </div>
        )}
        <div className="sm:col-span-2">
          <dt className="text-xs font-medium text-gray-500">Serviços Solicitados</dt>
          <dd className="mt-1 flex flex-wrap gap-1.5">
            {requested.length === 0 ? (
              <span className="text-sm text-gray-500">Nenhum</span>
            ) : (
              requested.map((label) => (
                <span
                  key={label}
                  className="rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700"
                >
                  {label}
                </span>
              ))
            )}
          </dd>
        </div>
        {triage.notes && (
          <div className="sm:col-span-2">
            <dt className="text-xs font-medium text-gray-500">Observações</dt>
            <dd className="mt-1 whitespace-pre-wrap text-sm text-gray-900">{triage.notes}</dd>
          </div>
        )}
      </dl>
    </div>
  );
}
