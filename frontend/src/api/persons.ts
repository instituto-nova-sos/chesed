import { apiClient, apiClientRaw } from './client';
import type {
  PersonListResponse,
  PersonDetail,
  Person,
  DuplicateCheckResult,
  PersonRole,
  VolunteerAgreement,
  HistoryEntry,
  CreatePersonInput,
  UpdatePersonInput,
  AddRoleInput,
  SelfRegisterInput,
} from '../types';

export function listPersons(params: {
  q?: string;
  page?: number;
  per_page?: number;
  agreement_status?: string;
}): Promise<PersonListResponse> {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set('q', params.q);
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  if (params.agreement_status) searchParams.set('agreement_status', params.agreement_status);
  const qs = searchParams.toString();
  return apiClient<PersonListResponse>(`/persons${qs ? `?${qs}` : ''}`);
}

export function getPerson(id: string): Promise<PersonDetail> {
  return apiClient<PersonDetail>(`/persons/${id}`);
}

export function createPerson(input: CreatePersonInput): Promise<Person> {
  return apiClient<Person>('/persons', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updatePerson(
  id: string,
  input: UpdatePersonInput,
): Promise<Person> {
  return apiClient<Person>(`/persons/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export function checkDuplicate(
  documentType: string,
  documentNumber: string,
): Promise<DuplicateCheckResult> {
  const params = new URLSearchParams({
    document_type: documentType,
    document_number: documentNumber,
  });
  return apiClient<DuplicateCheckResult>(
    `/persons/check-duplicate?${params.toString()}`,
  );
}

export function addPersonRole(
  personId: string,
  input: AddRoleInput,
): Promise<PersonRole> {
  return apiClient<PersonRole>(`/persons/${personId}/roles`, {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function togglePersonRole(
  personId: string,
  roleId: string,
  isActive: boolean,
): Promise<PersonRole> {
  return apiClient<PersonRole>(`/persons/${personId}/roles/${roleId}`, {
    method: 'PATCH',
    body: JSON.stringify({ is_active: isActive }),
  });
}

export function selfRegister(input: SelfRegisterInput): Promise<Person> {
  return apiClient<Person>('/self-register', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function getPersonHistory(
  personId: string,
): Promise<{ person: { id: string; full_name: string }; history: HistoryEntry[] }> {
  return apiClient(`/persons/${personId}/history`);
}

// --- Volunteer Agreement ---

export function getAgreementText(version?: string): Promise<{ text: string; version: string }> {
  const qs = version ? `?version=${version}` : '';
  return apiClient(`/volunteer-agreement/text${qs}`);
}

export function acceptAgreement(signatureData: string): Promise<VolunteerAgreement> {
  return apiClient<VolunteerAgreement>('/volunteer-agreement/accept', {
    method: 'POST',
    body: JSON.stringify({ signature_data: signatureData }),
  });
}

/** Self-service upload: the volunteer attaches their own signed agreement (image/PDF). */
export function uploadAgreementSelf(file: File): Promise<VolunteerAgreement> {
  const formData = new FormData();
  formData.append('document', file);
  return apiClientRaw<VolunteerAgreement>('/volunteer-agreement/upload', {
    method: 'POST',
    body: formData,
  });
}

export function rejectAgreement(reason?: string): Promise<VolunteerAgreement> {
  return apiClient<VolunteerAgreement>('/volunteer-agreement/reject', {
    method: 'POST',
    body: JSON.stringify({ reason }),
  });
}

export function getPersonAgreement(
  personId: string,
): Promise<{ agreements: VolunteerAgreement[] }> {
  return apiClient(`/persons/${personId}/agreement`);
}

export function uploadAgreement(
  personId: string,
  file: File,
): Promise<VolunteerAgreement> {
  const formData = new FormData();
  formData.append('document', file);
  return apiClientRaw<VolunteerAgreement>(`/persons/${personId}/agreement/upload`, {
    method: 'POST',
    body: formData,
  });
}

export function downloadAgreementDocument(personId: string): string {
  const baseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1';
  return `${baseUrl}/persons/${personId}/agreement/document`;
}
