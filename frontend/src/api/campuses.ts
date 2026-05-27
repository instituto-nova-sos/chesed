import { apiClient } from './client';

export interface Campus {
  id: string;
  name: string;
  region: string;
  city: string | null;
  state: string | null;
  country: string;
  is_active: boolean;
}

export interface CampusInput {
  name: string;
  region: string;
  city?: string;
  state?: string;
  country: string;
  is_active?: boolean;
}

export function listActiveCampuses(): Promise<{ data: Campus[] }> {
  return apiClient<{ data: Campus[] }>('/campuses');
}

export function listAllCampuses(): Promise<{ data: Campus[] }> {
  return apiClient<{ data: Campus[] }>('/campuses/all');
}

export function getCampus(id: string): Promise<Campus> {
  return apiClient<Campus>(`/campuses/${id}`);
}

export function createCampus(input: CampusInput): Promise<Campus> {
  return apiClient<Campus>('/campuses', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateCampus(id: string, input: CampusInput): Promise<Campus> {
  return apiClient<Campus>(`/campuses/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}
