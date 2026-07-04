import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { ApiError } from '../../../api/client';
import type { PersonDocument } from '../../../types';

const uploadPersonDocument = vi.fn();
vi.mock('../../../api/documents', () => ({
  uploadPersonDocument: (...args: unknown[]) => uploadPersonDocument(...args),
}));

import { DocumentUploadModal } from '../DocumentUploadModal';

const PERSON_ID = '00000000-0000-0000-0000-0000000000d1';

const createdDoc: PersonDocument = {
  id: '00000000-0000-0000-0000-0000000000d2',
  person_id: PERSON_ID,
  attendance_id: null,
  document_type: 'EXAM',
  file_name: 'laudo.pdf',
  file_size: 1024,
  mime_type: 'application/pdf',
  description: null,
  uploaded_at: '2026-07-03T14:30:00Z',
};

function renderModal(onUploaded = vi.fn(), onClose = vi.fn()) {
  render(<DocumentUploadModal personId={PERSON_ID} onClose={onClose} onUploaded={onUploaded} />);
  return { onUploaded, onClose };
}

function selectFile(file: File) {
  fireEvent.change(screen.getByLabelText('Arquivo do documento'), { target: { files: [file] } });
}

function pdfFile(name = 'laudo.pdf'): File {
  return new File(['%PDF-'], name, { type: 'application/pdf' });
}

function chooseType(value: string) {
  fireEvent.change(screen.getByLabelText('Tipo de documento'), { target: { value } });
}

function submitUpload(file: File, type: string) {
  selectFile(file);
  chooseType(type);
  fireEvent.click(screen.getByRole('button', { name: 'Enviar' }));
}

describe('DocumentUploadModal', () => {
  beforeEach(() => {
    uploadPersonDocument.mockReset();
  });

  it('rejects a disallowed file type with a pt-BR message', () => {
    renderModal();
    selectFile(new File(['plain'], 'nota.txt', { type: 'text/plain' }));
    expect(screen.getByText('Formato inválido. Aceitos: PDF, JPEG, PNG.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Enviar' })).toBeDisabled();
  });

  it('rejects an oversize file with a pt-BR message', () => {
    renderModal();
    const big = pdfFile('grande.pdf');
    Object.defineProperty(big, 'size', { value: 11 * 1024 * 1024 });
    selectFile(big);
    expect(screen.getByText('Arquivo excede o limite de 10MB.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Enviar' })).toBeDisabled();
  });

  it('keeps submit disabled until both file and type are chosen', () => {
    renderModal();
    expect(screen.getByRole('button', { name: 'Enviar' })).toBeDisabled();
    selectFile(pdfFile());
    expect(screen.getByRole('button', { name: 'Enviar' })).toBeDisabled();
    chooseType('EXAM');
    expect(screen.getByRole('button', { name: 'Enviar' })).toBeEnabled();
  });

  it('uploads with the selected type and description and notifies the parent', async () => {
    uploadPersonDocument.mockResolvedValue(createdDoc);
    const { onUploaded } = renderModal();

    const file = pdfFile();
    selectFile(file);
    chooseType('EXAM');
    fireEvent.change(screen.getByLabelText('Descrição (opcional)'), {
      target: { value: 'Resultado de exame' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Enviar' }));

    await waitFor(() =>
      expect(uploadPersonDocument).toHaveBeenCalledWith(
        PERSON_ID,
        file,
        'EXAM',
        'Resultado de exame',
      ),
    );
    await waitFor(() => expect(onUploaded).toHaveBeenCalledWith(createdDoc));
  });

  it('omits the description when it is empty', async () => {
    uploadPersonDocument.mockResolvedValue(createdDoc);
    renderModal();

    submitUpload(pdfFile(), 'ID');

    await waitFor(() =>
      expect(uploadPersonDocument).toHaveBeenCalledWith(
        PERSON_ID,
        expect.any(File),
        'ID',
        undefined,
      ),
    );
  });

  it('maps a 400 validation error to a readable pt-BR message', async () => {
    uploadPersonDocument.mockRejectedValue(new ApiError(400, { error: 'validation_error' }));
    const { onUploaded } = renderModal();

    submitUpload(pdfFile(), 'EXAM');

    expect(
      await screen.findByText('Arquivo ou tipo de documento inválido.'),
    ).toBeInTheDocument();
    expect(onUploaded).not.toHaveBeenCalled();
  });

  it('shows a generic pt-BR message for unexpected errors', async () => {
    uploadPersonDocument.mockRejectedValue(new ApiError(500, { error: 'internal_error' }));
    renderModal();

    submitUpload(pdfFile(), 'EXAM');

    expect(
      await screen.findByText('Erro ao enviar documento. Tente novamente.'),
    ).toBeInTheDocument();
  });
});
