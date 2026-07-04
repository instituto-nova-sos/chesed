import { useState } from 'react';
import { getDocumentDownloadUrl } from '../../api/documents';
import { documentTypeLabel } from '../../utils/documentLabels';
import { Alert } from '../ui/Alert';
import type { PersonDocument } from '../../types';

interface DocumentListProps {
  documents: PersonDocument[];
  disabled?: boolean;
}

export function DocumentList({ documents, disabled = false }: DocumentListProps) {
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleDownload = (documentId: string) => {
    setDownloadingId(documentId);
    setError(null);
    getDocumentDownloadUrl(documentId)
      .then(({ url }) => {
        window.open(url, '_blank', 'noopener');
      })
      .catch(() => {
        setError('Não foi possível gerar o link de download.');
      })
      .finally(() => {
        setDownloadingId(null);
      });
  };

  if (documents.length === 0) {
    return <p className="text-sm text-gray-500">Nenhum documento cadastrado.</p>;
  }

  return (
    <div className="space-y-3">
      {error && <Alert variant="error">{error}</Alert>}
      <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
        {documents.map((item) => (
          <li
            key={item.id}
            className="flex flex-wrap items-center justify-between gap-2 px-4 py-3"
          >
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-gray-900">{item.file_name}</p>
              <p className="text-xs text-gray-500">
                {documentTypeLabel(item.document_type)} ·{' '}
                {new Date(item.uploaded_at).toLocaleDateString('pt-BR')}
              </p>
            </div>
            <button
              type="button"
              onClick={() => handleDownload(item.id)}
              disabled={disabled || downloadingId === item.id}
              className="text-sm font-medium text-blue-600 hover:text-blue-800 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Baixar
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
