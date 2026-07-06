import { useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { Alert } from '../components/ui/Alert';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Pagination } from '../components/ui/Pagination';
import {
  AuditLogFilters,
  type AuditFilterValues,
} from '../components/audit/AuditLogFilters';
import { AuditLogTable } from '../components/audit/AuditLogTable';
import { useAuditLogs } from '../hooks/useAuditLogs';
import { useOfflineStatus } from '../hooks/useOfflineStatus';
import type { AuditLogFilters as AuditLogQuery } from '../types/audit';

const PER_PAGE = 50;

function defaultRange(): { start: string; end: string } {
  const today = new Date();
  const end = today.toISOString().slice(0, 10);
  const first = new Date(today.getFullYear(), today.getMonth(), 1)
    .toISOString()
    .slice(0, 10);
  return { start: first, end };
}

function buildQuery(values: AuditFilterValues, page: number): AuditLogQuery {
  return {
    start: values.start,
    end: values.end,
    user_id: values.user_id || undefined,
    entity_type: values.entity_type || undefined,
    action_type: values.action_type || undefined,
    page,
    per_page: PER_PAGE,
  };
}

export function AuditLogPage() {
  const { isOffline } = useOfflineStatus();
  const initialRange = useMemo(() => defaultRange(), []);
  const [values, setValues] = useState<AuditFilterValues>({
    ...initialRange,
    user_id: '',
    entity_type: '',
    action_type: '',
  });

  const initialQuery = useMemo(
    () => (isOffline ? undefined : buildQuery({ ...initialRange, user_id: '', entity_type: '', action_type: '' }, 1)),
    [initialRange, isOffline],
  );

  const { page, isLoading, error, query } = useAuditLogs(initialQuery);

  if (isOffline) {
    return (
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-gray-900">Registros de auditoria</h1>
        <Alert variant="info">
          Os registros de auditoria não estão disponíveis offline.
        </Alert>
      </div>
    );
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!values.start || !values.end) return;
    query(buildQuery(values, 1));
  }

  function handlePageChange(next: number) {
    query(buildQuery(values, next));
  }

  return (
    <div className="space-y-6">
      <h1 className="text-lg font-semibold text-gray-900">Registros de auditoria</h1>

      <AuditLogFilters
        values={values}
        onChange={(key, value) => setValues((v) => ({ ...v, [key]: value }))}
        onSubmit={handleSubmit}
        isLoading={isLoading}
      />

      {error && <Alert variant="error">{error}</Alert>}

      {isLoading && <LoadingScreen />}

      {!isLoading && page && page.data.length === 0 && (
        <EmptyState
          title="Nenhum registro"
          description="Nenhum registro de auditoria para os filtros selecionados."
        />
      )}

      {!isLoading && page && page.data.length > 0 && (
        <div className="space-y-3">
          <AuditLogTable rows={page.data} />
          <Pagination
            page={page.pagination.page}
            totalPages={page.pagination.total_pages}
            onPageChange={handlePageChange}
          />
        </div>
      )}
    </div>
  );
}
