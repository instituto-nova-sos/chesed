import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PersonDocument } from '../../../types';

const ROLE_HIERARCHY: Record<string, number> = {
  VOLUNTEER: 1,
  SECRETARY: 2,
  PROFESSIONAL: 3,
  COORDINATOR: 4,
  ADMIN: 5,
};

const authState = { role: 'VOLUNTEER' };
vi.mock('../../../hooks/useAuth', () => ({
  useAuth: () => ({
    hasMinRole: (minRole: string) =>
      (ROLE_HIERARCHY[authState.role] ?? 0) >= (ROLE_HIERARCHY[minRole] ?? 0),
  }),
}));

const offlineState = { isOffline: false };
vi.mock('../../../hooks/useOfflineStatus', () => ({
  useOfflineStatus: () => offlineState,
}));

const documentsState: {
  documents: PersonDocument[];
  isLoading: boolean;
  error: string | null;
} = { documents: [], isLoading: false, error: null };
const refresh = vi.fn((): Promise<void> => Promise.resolve());
const hookCalls: Array<string | undefined> = [];
vi.mock('../../../hooks/usePersonDocuments', () => ({
  usePersonDocuments: (personId: string | undefined) => {
    hookCalls.push(personId);
    return { ...documentsState, refresh };
  },
}));

const uploadPersonDocument = vi.fn();
vi.mock('../../../api/documents', () => ({
  uploadPersonDocument: (...args: unknown[]) => uploadPersonDocument(...args),
  getDocumentDownloadUrl: vi.fn(),
}));

import { PersonDocumentsSection } from '../PersonDocumentsSection';

function doc(overrides: Partial<PersonDocument> = {}): PersonDocument {
  return {
    id: '00000000-0000-0000-0000-0000000000d2',
    person_id: '00000000-0000-0000-0000-0000000000d1',
    attendance_id: null,
    document_type: 'PROOF_OF_RESIDENCE',
    file_name: 'comprovante.pdf',
    file_size: 182734,
    mime_type: 'application/pdf',
    description: null,
    uploaded_at: '2026-07-03T14:30:00Z',
    ...overrides,
  };
}

function renderSection() {
  return render(<PersonDocumentsSection personId="00000000-0000-0000-0000-0000000000d1" />);
}

describe('PersonDocumentsSection', () => {
  beforeEach(() => {
    authState.role = 'VOLUNTEER';
    offlineState.isOffline = false;
    documentsState.documents = [];
    documentsState.isLoading = false;
    documentsState.error = null;
    hookCalls.length = 0;
    refresh.mockClear();
    uploadPersonDocument.mockReset();
  });

  it('renders nothing for a VOLUNTEER', () => {
    const { container } = renderSection();
    expect(container).toBeEmptyDOMElement();
  });

  it('shows the upload button but hides the list for a SECRETARY', () => {
    authState.role = 'SECRETARY';
    documentsState.documents = [doc()];
    renderSection();
    expect(screen.getByRole('button', { name: 'Enviar documento' })).toBeInTheDocument();
    expect(screen.queryByText('comprovante.pdf')).not.toBeInTheDocument();
    expect(hookCalls).toContain(undefined);
    expect(hookCalls).not.toContain('00000000-0000-0000-0000-0000000000d1');
  });

  it('shows the document list for a PROFESSIONAL', () => {
    authState.role = 'PROFESSIONAL';
    documentsState.documents = [doc()];
    renderSection();
    expect(screen.getByText('comprovante.pdf')).toBeInTheDocument();
    expect(screen.getByText(/comprovante de residência/i)).toBeInTheDocument();
    expect(hookCalls).toContain('00000000-0000-0000-0000-0000000000d1');
  });

  it('shows an empty state for a PROFESSIONAL without documents', () => {
    authState.role = 'PROFESSIONAL';
    renderSection();
    expect(screen.getByText(/nenhum documento/i)).toBeInTheDocument();
  });

  it('surfaces the hook error', () => {
    authState.role = 'PROFESSIONAL';
    documentsState.error = 'Não foi possível carregar os documentos.';
    renderSection();
    expect(screen.getByText('Não foi possível carregar os documentos.')).toBeInTheDocument();
  });

  it('disables upload and shows the offline message when offline', () => {
    authState.role = 'SECRETARY';
    offlineState.isOffline = true;
    renderSection();
    expect(screen.getByRole('button', { name: 'Enviar documento' })).toBeDisabled();
    expect(screen.getByText('Conecte-se para gerenciar documentos.')).toBeInTheDocument();
  });

  it('uploads through the modal and refreshes the list on success', async () => {
    authState.role = 'PROFESSIONAL';
    uploadPersonDocument.mockResolvedValue(doc({ document_type: 'EXAM', file_name: 'laudo.pdf' }));
    renderSection();

    fireEvent.click(screen.getByRole('button', { name: 'Enviar documento' }));
    expect(screen.getByRole('heading', { name: 'Enviar Documento' })).toBeInTheDocument();

    const file = new File(['%PDF-'], 'laudo.pdf', { type: 'application/pdf' });
    fireEvent.change(screen.getByLabelText('Arquivo do documento'), {
      target: { files: [file] },
    });
    fireEvent.change(screen.getByLabelText('Tipo de documento'), {
      target: { value: 'EXAM' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Enviar' }));

    await waitFor(() => expect(uploadPersonDocument).toHaveBeenCalled());
    await waitFor(() => expect(refresh).toHaveBeenCalled());
    expect(screen.queryByRole('heading', { name: 'Enviar Documento' })).not.toBeInTheDocument();
  });
});
