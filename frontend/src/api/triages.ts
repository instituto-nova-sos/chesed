import { apiClient } from './client';
import type {
  Triage,
  TriageListResponse,
  CreateTriageInput,
  UpdateTriageInput,
} from '../types';

export interface ListTriagesParams {
  page?: number;
  per_page?: number;
  person_id?: string;
  from?: string;
  to?: string;
}

export function listTriages(params: ListTriagesParams = {}): Promise<TriageListResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  if (params.person_id) searchParams.set('person_id', params.person_id);
  if (params.from) searchParams.set('from', params.from);
  if (params.to) searchParams.set('to', params.to);
  const qs = searchParams.toString();
  return apiClient<TriageListResponse>(`/triages${qs ? `?${qs}` : ''}`);
}

export function getTriage(id: string): Promise<Triage> {
  return apiClient<Triage>(`/triages/${id}`);
}

export function createTriage(input: CreateTriageInput): Promise<Triage> {
  return apiClient<Triage>('/triages', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateTriage(id: string, input: UpdateTriageInput): Promise<Triage> {
  return apiClient<Triage>(`/triages/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  });
}
