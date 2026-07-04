import { useState } from 'react';
import { useAuth } from '../../hooks/useAuth';
import { useOfflineStatus } from '../../hooks/useOfflineStatus';
import { usePersonDocuments } from '../../hooks/usePersonDocuments';
import { Alert } from '../ui/Alert';
import { Button } from '../ui/Button';
import { DocumentList } from './DocumentList';
import { DocumentUploadModal } from './DocumentUploadModal';
import type { PersonDocument } from '../../types';

interface PersonDocumentsSectionProps {
  personId: string;
}

interface DocumentsPanelProps {
  isLoading: boolean;
  error: string | null;
  documents: PersonDocument[];
  disabled: boolean;
}

function DocumentsPanel({ isLoading, error, documents, disabled }: DocumentsPanelProps) {
  if (isLoading) return <div className="h-6 w-32 animate-pulse rounded bg-gray-200" />;
  if (error) return <p className="text-sm text-red-600">{error}</p>;
  return <DocumentList documents={documents} disabled={disabled} />;
}

export function PersonDocumentsSection({ personId }: PersonDocumentsSectionProps) {
  const { hasMinRole } = useAuth();
  const { isOffline } = useOfflineStatus();
  const canUpload = hasMinRole('SECRETARY');
  // Listing/downloading requires PROFESSIONAL+ (docs/16); SECRETARY only uploads,
  // so the list fetch is disabled below that role instead of returning 403.
  const canView = hasMinRole('PROFESSIONAL');
  const { documents, isLoading, error, refresh } = usePersonDocuments(
    canView ? personId : undefined,
  );
  const [showUpload, setShowUpload] = useState(false);

  if (!canUpload) return null;

  return (
    <div className="space-y-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-medium text-gray-900">Documentos</h3>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => setShowUpload(true)}
          disabled={isOffline}
        >
          Enviar documento
        </Button>
      </div>

      {isOffline && <Alert variant="warning">Conecte-se para gerenciar documentos.</Alert>}

      {canView && (
        <DocumentsPanel
          isLoading={isLoading}
          error={error}
          documents={documents}
          disabled={isOffline}
        />
      )}

      {showUpload && (
        <DocumentUploadModal
          personId={personId}
          onClose={() => setShowUpload(false)}
          onUploaded={() => {
            setShowUpload(false);
            void refresh();
          }}
        />
      )}
    </div>
  );
}
