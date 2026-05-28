export interface Person {
  id: string;
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  gender?: string;
  email?: string;
  phone?: string;
  photo_url?: string;
  referral_source?: string;
  nationality: string;
  campus_id: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface PersonListItem {
  id: string;
  full_name: string;
  document_number?: string;
  phone?: string;
  roles: string[];
  is_active: boolean;
}

export interface Address {
  id: string;
  person_id: string;
  street?: string;
  number?: string;
  complement?: string;
  neighborhood?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  country: string;
  is_primary: boolean;
  created_at: string;
  updated_at: string;
}

export interface PersonRole {
  id: string;
  person_id: string;
  role_type: string;
  professional_specialty?: string;
  is_active: boolean;
  activated_at: string;
  deactivated_at?: string;
  activated_by?: string;
  deactivated_by?: string;
  notes?: string;
}

export interface VolunteerAgreement {
  id: string;
  person_id: string;
  person_role_id: string;
  campus_id: string;
  status: 'PENDING' | 'ACCEPTED' | 'REJECTED';
  signature_method?: 'DIGITAL' | 'MANUAL_UPLOAD';
  accepted_at?: string;
  accepted_by_user?: string;
  ip_address?: string;
  user_agent?: string;
  document_path?: string;
  uploaded_at?: string;
  uploaded_by?: string;
  rejected_at?: string;
  rejection_reason?: string;
  agreement_version: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface PersonDetail extends Person {
  addresses: Address[];
  roles: PersonRole[];
}

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface PersonListResponse {
  data: PersonListItem[];
  pagination: Pagination;
}

export interface DuplicateMatch {
  id: string;
  full_name: string;
  document_number: string;
  campus: string;
  match_type: string;
}

export interface DuplicateCheckResult {
  has_duplicates: boolean;
  matches: DuplicateMatch[];
}

export interface HistoryEntry {
  type: string;
  id: string;
  date: string;
  summary: string;
  service_type?: string;
  professional?: string;
  status?: string;
  campaign?: string;
}

export interface CreatePersonInput {
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  nationality?: string;
  gender?: string;
  email?: string;
  phone?: string;
  referral_source?: string;
  address?: AddressInput;
  sync_id?: string;
}

export interface UpdatePersonInput {
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  nationality?: string;
  gender?: string;
  email?: string;
  phone?: string;
  referral_source?: string;
  address?: AddressInput;
}

export interface SelfRegisterInput {
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  nationality?: string;
  gender?: string;
  phone?: string;
  referral_source?: string;
  address?: AddressInput;
  role_type: 'VOLUNTEER' | 'ASSISTED';
  campus_id: string;
}

export interface AddressInput {
  street?: string;
  number?: string;
  complement?: string;
  neighborhood?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  country?: string;
}

export interface AddRoleInput {
  role_type: string;
  professional_specialty?: string;
  notes?: string;
}
