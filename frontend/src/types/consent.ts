export type ConsentType =
  | 'DATA_PROCESSING'
  | 'IMAGE_USAGE'
  | 'HEALTH_DATA'
  | 'MINOR_GUARDIAN';

export interface Consent {
  id: string;
  person_id: string;
  consent_type: ConsentType;
  consent_version: string;
  purpose: string;
  granted_at: string;
  granted_by_person_id: string | null;
  signature_data: string | null;
  is_active: boolean;
  revoked_at: string | null;
  revoked_reason: string | null;
}

export interface CreateConsentInput {
  person_id: string;
  consent_type: ConsentType;
  purpose: string;
  consent_version?: string;
  granted_by_person_id?: string;
  signature_data?: string;
}

export interface ConsentListResponse {
  data: Consent[];
}
