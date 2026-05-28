import type { Pagination } from './person';

export interface Triage {
  id: string;
  person_id: string;
  campus_id: string;
  main_complaint: string;
  assigned_team?: string;
  triage_date: string;
  location?: string;
  triaged_by: string;
  notes?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  requested_service_types?: string[];
}

export interface TriageListItem {
  id: string;
  person_id: string;
  person_name: string;
  main_complaint: string;
  triage_date: string;
  requested_service_count: number;
}

export interface TriageListResponse {
  data: TriageListItem[];
  pagination: Pagination;
}

export interface CreateTriageInput {
  person_id: string;
  main_complaint: string;
  assigned_team?: string;
  triage_date?: string;
  location?: string;
  notes?: string;
  requested_service_types?: string[];
}

export type UpdateTriageInput = Omit<CreateTriageInput, 'person_id'>;
