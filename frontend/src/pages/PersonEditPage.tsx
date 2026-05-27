/**
 * Offline Behavior: Category C — Online-Only
 * - Edit requires fetching latest data from server
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useNavigate, useParams } from 'react-router-dom';
import { usePerson } from '../hooks/usePerson';
import { Button } from '../components/ui/Button';
import { PersonForm } from '../components/person/PersonForm';
import { LoadingScreen } from '../components/ui/LoadingScreen';

export function PersonEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { person, isLoading, error } = usePerson(id);

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
    <div className="mx-auto max-w-2xl space-y-4">
      <div className="flex items-center gap-3">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => navigate(`/persons/${id}`)}
        >
          Voltar
        </Button>
        <h1 className="text-lg font-semibold text-gray-900">
          Editar: {person.full_name}
        </h1>
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:p-6">
        <PersonForm editData={person} />
      </div>
    </div>
  );
}
