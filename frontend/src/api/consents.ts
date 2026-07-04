import { apiClient } from './client';
import type { Consent, ConsentListResponse, CreateConsentInput } from '../types/consent';

export function createConsent(input: CreateConsentInput): Promise<Consent> {
  return apiClient<Consent>('/consents', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function listPersonConsents(personId: string): Promise<ConsentListResponse> {
  return apiClient<ConsentListResponse>(`/persons/${personId}/consents`);
}

export function revokeConsent(consentId: string, reason: string): Promise<Consent> {
  return apiClient<Consent>(`/consents/${consentId}/revoke`, {
    method: 'PATCH',
    body: JSON.stringify({ revoked_reason: reason }),
  });
}
