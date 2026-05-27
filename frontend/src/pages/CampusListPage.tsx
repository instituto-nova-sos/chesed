import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { listAllCampuses, type Campus } from '../api/campuses';
import { useAuth } from '../hooks/useAuth';
import { Badge } from '../components/ui/Badge';

export function CampusListPage() {
  const { hasRole } = useAuth();
  const [campuses, setCampuses] = useState<Campus[]>([]);
  const [loading, setLoading] = useState(true);
  const isAdmin = hasRole('ADMIN');

  useEffect(() => {
    (isAdmin ? listAllCampuses() : Promise.resolve({ data: [] }))
      .then(({ data }) => setCampuses(data))
      .catch(() => setCampuses([]))
      .finally(() => setLoading(false));
  }, [isAdmin]);

  if (!isAdmin) {
    return (
      <div className="p-6 text-center text-gray-500">
        Acesso restrito a administradores.
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900">Campus</h1>
        <Link
          to="/campuses/new"
          className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Novo Campus
        </Link>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      ) : campuses.length === 0 ? (
        <p className="text-center text-gray-500 py-12">
          Nenhum campus cadastrado.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Nome
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Região
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Cidade / Estado
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  País
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {campuses.map((campus) => (
                <tr key={campus.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">
                    {campus.name}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {campus.region}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {[campus.city, campus.state].filter(Boolean).join(', ') ||
                      '—'}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {campus.country}
                  </td>
                  <td className="px-4 py-3">
                    <Badge
                      variant={campus.is_active ? 'success' : 'neutral'}
                    >
                      {campus.is_active ? 'Ativo' : 'Inativo'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link
                      to={`/campuses/${campus.id}/edit`}
                      className="text-sm text-blue-600 hover:text-blue-800"
                    >
                      Editar
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
