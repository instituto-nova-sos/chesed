import { useState, useRef } from 'react';
import { uploadAgreement } from '../../api/persons';
import { Button } from '../ui/Button';
import { Alert } from '../ui/Alert';
import type { VolunteerAgreement } from '../../types';

interface AgreementUploadModalProps {
  personId: string;
  onClose: () => void;
  onUploaded: (agreement: VolunteerAgreement) => void;
}

const ACCEPTED_TYPES = ['application/pdf', 'image/jpeg', 'image/png'];
const MAX_SIZE_MB = 10;

export function AgreementUploadModal({ personId, onClose, onUploaded }: AgreementUploadModalProps) {
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setError(null);
    const selected = e.target.files?.[0];
    if (!selected) return;

    if (!ACCEPTED_TYPES.includes(selected.type)) {
      setError('Formato invalido. Aceitos: PDF, JPEG, PNG.');
      return;
    }
    if (selected.size > MAX_SIZE_MB * 1024 * 1024) {
      setError(`Arquivo excede o limite de ${MAX_SIZE_MB}MB.`);
      return;
    }

    setFile(selected);
  };

  const handleUpload = async () => {
    if (!file) return;
    setUploading(true);
    setError(null);

    try {
      const agreement = await uploadAgreement(personId, file);
      onUploaded(agreement);
    } catch {
      setError('Erro ao enviar documento. Tente novamente.');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Enviar Termo Assinado
        </h3>

        <p className="text-sm text-gray-600 mb-4">
          Selecione o documento digitalizado do termo de voluntario assinado.
          Formatos aceitos: PDF, JPEG, PNG (max {MAX_SIZE_MB}MB).
        </p>

        <input
          ref={inputRef}
          type="file"
          accept=".pdf,.jpg,.jpeg,.png"
          onChange={handleFileChange}
          className="block w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
        />

        {file && (
          <p className="mt-2 text-sm text-gray-600">
            Selecionado: {file.name} ({(file.size / 1024 / 1024).toFixed(2)}MB)
          </p>
        )}

        {error && (
          <div className="mt-3">
            <Alert type="error">{error}</Alert>
          </div>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={onClose} disabled={uploading}>
            Cancelar
          </Button>
          <Button onClick={handleUpload} disabled={!file || uploading}>
            {uploading ? 'Enviando...' : 'Enviar'}
          </Button>
        </div>
      </div>
    </div>
  );
}
