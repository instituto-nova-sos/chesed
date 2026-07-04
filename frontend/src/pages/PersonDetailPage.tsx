/**
 * Offline Behavior: Category B — Read-Only Offline (Cached Data)
 * - Displays cached person detail when offline
 * - Role management disabled when offline
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useNavigate, useParams } from 'react-router-dom';
import { usePerson } from '../hooks/usePerson';
import { Button } from '../components/ui/Button';
import { PersonInfo } from '../components/person/PersonInfo';
import { RoleBadgeList } from '../components/person/RoleBadgeList';
import { AgreementStatusCard } from '../components/person/AgreementStatusCard';
import { PersonDocumentsSection } from '../components/documents/PersonDocumentsSection';
import { PersonConsentsSection } from '../components/consents/PersonConsentsSection';
import { LoadingScreen } from '../components/ui/LoadingScreen';

export function PersonDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { person, isLoading, error, refresh } = usePerson(id);

  if (isLoading) return <LoadingScreen />;

  if (error || !person) {
    return (
      <div className="py-8 text-center">
        <p className="text-sm text-red-600">
          {error || 'Pessoa não encontrada.'}
        </p>
        <Button
          variant="secondary"
          size="sm"
          className="mt-4"
          onClick={() => navigate('/persons')}
        >
          Voltar para lista
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="secondary" size="sm" onClick={() => navigate('/persons')}>
          Voltar
        </Button>
        <h1 className="text-lg font-semibold text-gray-900">
          Detalhes da Pessoa
        </h1>
        <div className="ml-auto flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate(`/triages/new?person_id=${person.id}`)}
          >
            Nova Triagem
          </Button>
          <Button
            size="sm"
            onClick={() => navigate(`/attendances/new?person_id=${person.id}`)}
          >
            Novo Atendimento
          </Button>
        </div>
      </div>

      <PersonInfo person={person} />

      <RoleBadgeList
        personId={person.id}
        roles={person.roles}
        onRolesChanged={refresh}
      />

      <AgreementStatusCard
        personId={person.id}
        hasVolunteerRole={person.roles.some((r) => r.role_type === 'VOLUNTEER' && r.is_active)}
      />

      <PersonDocumentsSection personId={person.id} />

      <PersonConsentsSection personId={person.id} />

      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h3 className="text-sm font-medium text-gray-900">
          Histórico de Atendimentos
        </h3>
        <p className="mt-2 text-sm text-gray-500">
          Nenhum histórico encontrado.
        </p>
      </div>
    </div>
  );
}
