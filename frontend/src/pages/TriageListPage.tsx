import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { useTriages } from '../hooks/useTriages';
import { EmptyState } from '../components/ui/EmptyState';
import { Pagination } from '../components/ui/Pagination';
import { LoadingScreen } from '../components/ui/LoadingScreen';

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function TriageListPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const personID = searchParams.get('person_id') ?? undefined;
  const { triages, pagination, isLoading, error, goToPage } = useTriages({
    person_id: personID,
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-900">Triagens</h1>
      </div>

      {error && <p className="text-center text-sm text-red-600">{error}</p>}

      {isLoading ? (
        <LoadingScreen />
      ) : triages.length === 0 ? (
        <EmptyState
          title="Nenhuma triagem encontrada"
          description="Triagens são criadas a partir do cadastro de uma pessoa."
        />
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-2 text-left font-medium text-gray-600">Pessoa</th>
                  <th className="px-4 py-2 text-left font-medium text-gray-600">Queixa</th>
                  <th className="px-4 py-2 text-left font-medium text-gray-600">Serviços</th>
                  <th className="px-4 py-2 text-left font-medium text-gray-600">Data</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {triages.map((t) => (
                  <tr
                    key={t.id}
                    className="cursor-pointer hover:bg-gray-50"
                    onClick={() => navigate(`/triages/${t.id}`)}
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/persons/${t.person_id}`}
                        className="text-blue-600 hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {t.person_name}
                      </Link>
                    </td>
                    <td className="max-w-md truncate px-4 py-3 text-gray-700">
                      {t.main_complaint}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{t.requested_service_count}</td>
                    <td className="px-4 py-3 text-gray-600">{formatDate(t.triage_date)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
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
