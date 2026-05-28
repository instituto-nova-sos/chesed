/**
 * Offline Behavior: Category B — Read-Only Offline (Cached Data)
 * - Displays cached person list when offline
 * - Search operates on cached data when offline
 * - "Nova Pessoa" available offline (Category A form)
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useNavigate } from 'react-router-dom';
import { usePersons } from '../hooks/usePersons';
import { useAuth } from '../hooks/useAuth';
import { SearchBar } from '../components/ui/SearchBar';
import { Pagination } from '../components/ui/Pagination';
import { EmptyState } from '../components/ui/EmptyState';
import { Button } from '../components/ui/Button';
import { PersonCard } from '../components/person/PersonCard';
import { LoadingScreen } from '../components/ui/LoadingScreen';

const agreementFilterOptions = [
  { value: '', label: 'Todos' },
  { value: 'with_agreement', label: 'Com Termo' },
  { value: 'without_agreement', label: 'Sem Termo' },
  { value: 'rejected', label: 'Recusaram' },
];

export function PersonListPage() {
  const navigate = useNavigate();
  const { hasMinRole } = useAuth();
  const {
    persons,
    pagination,
    isLoading,
    error,
    query,
    agreementStatus,
    search,
    filterByAgreement,
    goToPage,
  } = usePersons();

  const canFilterAgreement = hasMinRole('COORDINATOR');

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Pessoas</h1>
        <Button onClick={() => navigate('/persons/new')}>Nova Pessoa</Button>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="flex-1">
          <SearchBar value={query} onChange={search} />
        </div>
        {canFilterAgreement && (
          <div className="flex gap-1">
            {agreementFilterOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => filterByAgreement(opt.value)}
                className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                  agreementStatus === opt.value
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        )}
      </div>

      {error && (
        <p className="text-center text-sm text-red-600">{error}</p>
      )}

      {isLoading ? (
        <LoadingScreen />
      ) : persons.length === 0 ? (
        <EmptyState
          title="Nenhuma pessoa encontrada"
          description={
            query
              ? 'Tente buscar com outro termo.'
              : 'Cadastre a primeira pessoa clicando no botão acima.'
          }
        />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {persons.map((person) => (
              <PersonCard key={person.id} person={person} />
            ))}
          </div>

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
