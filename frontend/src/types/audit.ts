import type { Pagination } from './person';

export interface AuditLogItem {
  id: string;
  user_email: string | null;
  action_type: string;
  entity_type: string;
  entity_id: string | null;
  description: string | null;
  old_values: Record<string, unknown> | null;
  new_values: Record<string, unknown> | null;
  ip_address: string | null;
  timestamp: string;
}

export interface AuditLogFilters {
  start: string;
  end: string;
  user_id?: string;
  entity_type?: string;
  action_type?: string;
  page?: number;
  per_page?: number;
}

export interface AuditLogPage {
  data: AuditLogItem[];
  pagination: Pagination;
}
