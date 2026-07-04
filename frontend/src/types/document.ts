export type DocumentType =
  | 'ID'
  | 'PROOF_OF_RESIDENCE'
  | 'MEDICAL_RECORD'
  | 'EXAM'
  | 'CONSENT'
  | 'PHOTO'
  | 'OTHER';

export interface PersonDocument {
  id: string;
  person_id: string;
  attendance_id: string | null;
  document_type: DocumentType;
  file_name: string;
  file_size: number;
  mime_type: string;
  description: string | null;
  uploaded_at: string;
}

export interface PersonDocumentListResponse {
  data: PersonDocument[];
}

export interface DocumentDownload {
  url: string;
  expires_at: string;
}
