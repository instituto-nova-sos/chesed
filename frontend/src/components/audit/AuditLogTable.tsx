import type { AuditLogItem } from '../../types/audit';

function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString('pt-BR');
}

export function AuditLogTable({ rows }: { rows: AuditLogItem[] }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white shadow-sm">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50">
          <tr className="text-left text-xs uppercase text-gray-500">
            <th className="px-3 py-2 font-medium">Data/hora</th>
            <th className="px-3 py-2 font-medium">Usuário</th>
            <th className="px-3 py-2 font-medium">Ação</th>
            <th className="px-3 py-2 font-medium">Entidade</th>
            <th className="px-3 py-2 font-medium">Descrição</th>
            <th className="px-3 py-2 font-medium">IP</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {rows.map((row) => (
            <tr key={row.id} className="text-gray-700">
              <td className="whitespace-nowrap px-3 py-2">
                {formatTimestamp(row.timestamp)}
              </td>
              <td className="px-3 py-2">{row.user_email ?? '—'}</td>
              <td className="px-3 py-2 font-medium text-gray-900">
                {row.action_type}
              </td>
              <td className="px-3 py-2">{row.entity_type}</td>
              <td className="px-3 py-2">{row.description ?? '—'}</td>
              <td className="whitespace-nowrap px-3 py-2">
                {row.ip_address ?? '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
