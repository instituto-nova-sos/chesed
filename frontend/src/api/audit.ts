import { apiClient } from './client';
import type { AuditLogFilters, AuditLogPage } from '../types/audit';

export function listAuditLogs(filters: AuditLogFilters): Promise<AuditLogPage> {
  const search = new URLSearchParams({ start: filters.start, end: filters.end });
  if (filters.user_id) search.set('user_id', filters.user_id);
  if (filters.entity_type) search.set('entity_type', filters.entity_type);
  if (filters.action_type) search.set('action_type', filters.action_type);
  if (filters.page) search.set('page', String(filters.page));
  if (filters.per_page) search.set('per_page', String(filters.per_page));
  return apiClient<AuditLogPage>(`/audit/logs?${search.toString()}`);
}
