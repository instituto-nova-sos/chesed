import { useAuth } from '../hooks/useAuth';

export function DashboardPage() {
  const { email, roles, campusId } = useAuth();

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-800">Dashboard</h1>
      <p className="mt-1 text-sm text-gray-500">Bem-vindo ao Chesed</p>

      <div className="mt-6 rounded-lg border bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-700">Informacoes do Usuario</h2>
        <dl className="mt-4 space-y-2">
          <div className="flex gap-2">
            <dt className="text-sm font-medium text-gray-500">Email:</dt>
            <dd className="text-sm text-gray-900">{email}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-sm font-medium text-gray-500">Perfil:</dt>
            <dd className="text-sm text-gray-900">{roles.join(', ')}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-sm font-medium text-gray-500">Campus:</dt>
            <dd className="text-sm text-gray-900">{campusId ?? 'N/A'}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
