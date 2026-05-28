import { apiClient } from './client';

export interface ServiceType {
  id: string;
  code: string;
  name: string;
  description?: string;
  is_active: boolean;
}

export function listServiceTypes(): Promise<{ data: ServiceType[] }> {
  return apiClient<{ data: ServiceType[] }>('/service-types');
}
