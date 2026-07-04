import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PersonDocument } from '../../../types';

const getDocumentDownloadUrl = vi.fn();
vi.mock('../../../api/documents', () => ({
  getDocumentDownloadUrl: (...args: unknown[]) => getDocumentDownloadUrl(...args),
}));

import { DocumentList } from '../DocumentList';

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

describe('DocumentList', () => {
  let openSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    getDocumentDownloadUrl.mockReset();
    openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
  });

  afterEach(() => {
    openSpy.mockRestore();
  });

  it('shows an empty state when there are no documents', () => {
    render(<DocumentList documents={[]} />);
    expect(screen.getByText(/nenhum documento/i)).toBeInTheDocument();
  });

  it('renders the pt-BR type label, file name, and pt-BR upload date', () => {
    const item = doc();
    render(<DocumentList documents={[item]} />);
    expect(screen.getByText('comprovante.pdf')).toBeInTheDocument();
    expect(screen.getByText(/comprovante de residência/i)).toBeInTheDocument();
    const expectedDate = new Date(item.uploaded_at).toLocaleDateString('pt-BR');
    expect(screen.getByText(new RegExp(expectedDate))).toBeInTheDocument();
  });

  it('opens the presigned URL in a new tab on download', async () => {
    const url = 'https://storage.example.com/chesed-docs/documents/x?X-Amz-Signature=abc';
    getDocumentDownloadUrl.mockResolvedValue({ url, expires_at: '2026-07-03T14:45:00Z' });
    const item = doc();
    render(<DocumentList documents={[item]} />);

    fireEvent.click(screen.getByRole('button', { name: 'Baixar' }));

    await waitFor(() => expect(getDocumentDownloadUrl).toHaveBeenCalledWith(item.id));
    await waitFor(() => expect(openSpy).toHaveBeenCalledWith(url, '_blank', 'noopener'));
  });

  it('disables the download action when disabled (offline)', () => {
    render(<DocumentList documents={[doc()]} disabled />);
    expect(screen.getByRole('button', { name: 'Baixar' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Baixar' }));
    expect(getDocumentDownloadUrl).not.toHaveBeenCalled();
  });

  it('shows a pt-BR message when the download link cannot be generated', async () => {
    getDocumentDownloadUrl.mockRejectedValue(new Error('boom'));
    render(<DocumentList documents={[doc()]} />);

    fireEvent.click(screen.getByRole('button', { name: 'Baixar' }));

    expect(
      await screen.findByText('Não foi possível gerar o link de download.'),
    ).toBeInTheDocument();
    expect(openSpy).not.toHaveBeenCalled();
  });
});
