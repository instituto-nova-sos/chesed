import { useState } from 'react';
import type { ChangeEvent } from 'react';
import { uploadPersonDocument } from '../../api/documents';
import { ApiError } from '../../api/client';
import { Alert } from '../ui/Alert';
import { Button } from '../ui/Button';
import { Select } from '../ui/Select';
import { DOCUMENT_TYPE_LABELS, isDocumentType } from '../../utils/documentLabels';
import type { DocumentType, PersonDocument } from '../../types';

interface DocumentUploadModalProps {
  personId: string;
  onClose: () => void;
  onUploaded: (uploaded: PersonDocument) => void;
}

const ACCEPTED_TYPES = ['application/pdf', 'image/jpeg', 'image/png'];
const MAX_SIZE_MB = 10;

const TYPE_OPTIONS = Object.entries(DOCUMENT_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}));

function validateFile(file: File): string | null {
  if (!ACCEPTED_TYPES.includes(file.type)) {
    return 'Formato inválido. Aceitos: PDF, JPEG, PNG.';
  }
  if (file.size > MAX_SIZE_MB * 1024 * 1024) {
    return `Arquivo excede o limite de ${MAX_SIZE_MB}MB.`;
  }
  return null;
}

function uploadErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 400) return 'Arquivo ou tipo de documento inválido.';
    if (err.status === 404) return 'Pessoa não encontrada.';
  }
  return 'Erro ao enviar documento. Tente novamente.';
}

interface FileFieldProps {
  file: File | null;
  onChange: (e: ChangeEvent<HTMLInputElement>) => void;
}

function FileField({ file, onChange }: FileFieldProps) {
  return (
    <div>
      <label htmlFor="document-file" className="block text-sm font-medium text-gray-700">
        Arquivo do documento
      </label>
      <input
        id="document-file"
        type="file"
        accept=".pdf,.jpg,.jpeg,.png"
        onChange={onChange}
        className="mt-1 block w-full text-sm text-gray-500 file:mr-4 file:rounded-md file:border-0 file:bg-blue-50 file:px-4 file:py-2 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
      />
      <p className="mt-1 text-xs text-gray-500">
        Formatos aceitos: PDF, JPEG, PNG (máx. {MAX_SIZE_MB}MB).
      </p>
      {file && (
        <p className="mt-2 text-sm text-gray-600">
          Selecionado: {file.name} ({(file.size / 1024 / 1024).toFixed(2)}MB)
        </p>
      )}
    </div>
  );
}

interface DescriptionFieldProps {
  value: string;
  onChange: (value: string) => void;
}

function DescriptionField({ value, onChange }: DescriptionFieldProps) {
  return (
    <div>
      <label htmlFor="document-description" className="block text-sm font-medium text-gray-700">
        Descrição (opcional)
      </label>
      <input
        id="document-description"
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    </div>
  );
}

export function DocumentUploadModal({ personId, onClose, onUploaded }: DocumentUploadModalProps) {
  const [file, setFile] = useState<File | null>(null);
  const [documentType, setDocumentType] = useState<DocumentType | ''>('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (!selected) return;
    const validationError = validateFile(selected);
    setFile(validationError ? null : selected);
    setError(validationError);
  };

  const handleTypeChange = (e: ChangeEvent<HTMLSelectElement>) => {
    setDocumentType(isDocumentType(e.target.value) ? e.target.value : '');
  };

  const handleUpload = async () => {
    if (!file || documentType === '') return;
    setUploading(true);
    setError(null);
    try {
      const uploaded = await uploadPersonDocument(
        personId,
        file,
        documentType,
        description.trim() || undefined,
      );
      onUploaded(uploaded);
    } catch (err) {
      setError(uploadErrorMessage(err));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h3 className="mb-4 text-lg font-semibold text-gray-900">Enviar Documento</h3>
        <div className="space-y-4">
          <FileField file={file} onChange={handleFileChange} />
          <Select
            id="document-type"
            label="Tipo de documento"
            options={TYPE_OPTIONS}
            placeholder="Selecione o tipo"
            value={documentType}
            onChange={handleTypeChange}
          />
          <DescriptionField value={description} onChange={setDescription} />
          {error && <Alert variant="error">{error}</Alert>}
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={onClose} disabled={uploading}>
            Cancelar
          </Button>
          <Button onClick={handleUpload} disabled={!file || documentType === '' || uploading}>
            {uploading ? 'Enviando...' : 'Enviar'}
          </Button>
        </div>
      </div>
    </div>
  );
}
