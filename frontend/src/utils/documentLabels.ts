import type { DocumentType } from '../types';

export const DOCUMENT_TYPE_LABELS: Record<DocumentType, string> = {
  ID: 'Documento de identidade',
  PROOF_OF_RESIDENCE: 'Comprovante de residência',
  MEDICAL_RECORD: 'Prontuário médico',
  EXAM: 'Exame',
  CONSENT: 'Consentimento',
  PHOTO: 'Foto',
  OTHER: 'Outro',
};

export function isDocumentType(value: string): value is DocumentType {
  return value in DOCUMENT_TYPE_LABELS;
}

export function documentTypeLabel(type: DocumentType): string {
  return DOCUMENT_TYPE_LABELS[type] ?? type;
}
