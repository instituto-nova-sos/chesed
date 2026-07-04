import { apiClient, apiClientRaw } from './client';
import type {
  DocumentDownload,
  DocumentType,
  PersonDocument,
  PersonDocumentListResponse,
} from '../types';

export function uploadPersonDocument(
  personId: string,
  file: File,
  documentType: DocumentType,
  description?: string,
): Promise<PersonDocument> {
  const formData = new FormData();
  formData.append('document', file);
  formData.append('document_type', documentType);
  if (description) formData.append('description', description);
  return apiClientRaw<PersonDocument>(`/persons/${personId}/documents`, {
    method: 'POST',
    body: formData,
  });
}

export function listPersonDocuments(personId: string): Promise<PersonDocumentListResponse> {
  return apiClient<PersonDocumentListResponse>(`/persons/${personId}/documents`);
}

export function getDocumentDownloadUrl(documentId: string): Promise<DocumentDownload> {
  return apiClient<DocumentDownload>(`/documents/${documentId}/download`);
}
