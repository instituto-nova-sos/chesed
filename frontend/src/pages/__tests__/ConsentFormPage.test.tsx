import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { ApiError } from '../../api/client';
import { PURPOSE_PRESETS } from '../../utils/consentLabels';
import type { Consent } from '../../types/consent';

const createConsent = vi.fn();
vi.mock('../../api/consents', () => ({
  createConsent: (...args: unknown[]) => createConsent(...args) as Promise<Consent>,
}));

const offlineState = { isOffline: false };
vi.mock('../../hooks/useOfflineStatus', () => ({
  useOfflineStatus: () => offlineState,
}));

const personsSearch = vi.fn();
vi.mock('../../hooks/usePersons', () => ({
  usePersons: () => ({
    persons: [{ id: 'g-1', full_name: 'Maria Responsável' }],
    isLoading: false,
    error: null,
    search: personsSearch,
    pagination: { page: 1, per_page: 20, total: 1, total_pages: 1 },
  }),
}));

// The pad's canvas contract is covered by SignaturePadCanvas.test.tsx; the
// page test stubs it so signing is a deterministic click in jsdom.
vi.mock('../../components/ui/SignaturePadCanvas', () => ({
  SignaturePadCanvas: (props: {
    onChange: (dataUrl: string | null) => void;
    disabled?: boolean;
  }) => (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => props.onChange('data:image/png;base64,STUB')}
    >
      stub-assinar
    </button>
  ),
}));

import { ConsentFormPage } from '../ConsentFormPage';

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/persons/p-1/consents/new']}>
      <Routes>
        <Route path="/persons/:personId/consents/new" element={<ConsentFormPage />} />
        <Route path="/persons/:personId" element={<div>person detail</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

async function sign() {
  await userEvent.click(screen.getByRole('button', { name: 'stub-assinar' }));
}

async function submit() {
  await userEvent.click(screen.getByRole('button', { name: 'Salvar' }));
}

beforeEach(() => {
  createConsent.mockReset();
  offlineState.isOffline = false;
});

describe('ConsentFormPage — fields and validation', () => {
  it('renders the consent fields with the purpose preset in Portuguese', () => {
    renderPage();

    expect(screen.getByText('Novo consentimento')).toBeInTheDocument();
    expect(screen.getByLabelText('Tipo de consentimento')).toBeInTheDocument();
    expect(screen.getByLabelText('Finalidade')).toHaveValue(PURPOSE_PRESETS.DATA_PROCESSING);
    expect(screen.getByLabelText('Versão')).toHaveValue('1.0');
    expect(screen.getByLabelText('Versão')).toHaveAttribute('readonly');
  });

  it('refreshes the purpose preset when the consent type changes', async () => {
    renderPage();

    await userEvent.selectOptions(
      screen.getByLabelText('Tipo de consentimento'),
      'Uso de imagem',
    );

    expect(screen.getByLabelText('Finalidade')).toHaveValue(PURPOSE_PRESETS.IMAGE_USAGE);
  });

  it('blocks submit without a signature', async () => {
    renderPage();

    await submit();

    expect(await screen.findByText('Assinatura é obrigatória')).toBeInTheDocument();
    expect(createConsent).not.toHaveBeenCalled();
  });

  it('blocks submit without a purpose', async () => {
    renderPage();

    await userEvent.clear(screen.getByLabelText('Finalidade'));
    await sign();
    await submit();

    expect(await screen.findByText('Campo obrigatório')).toBeInTheDocument();
    expect(createConsent).not.toHaveBeenCalled();
  });

  it('submits the consent with the signature and navigates back to the person', async () => {
    createConsent.mockResolvedValue({ id: 'c-1' });
    renderPage();

    await sign();
    await submit();

    await waitFor(() => expect(createConsent).toHaveBeenCalledTimes(1));
    const payload = createConsent.mock.calls[0]?.[0] as Record<string, unknown>;
    expect(payload).toMatchObject({
      person_id: 'p-1',
      consent_type: 'DATA_PROCESSING',
      purpose: PURPOSE_PRESETS.DATA_PROCESSING,
      consent_version: '1.0',
      signature_data: 'data:image/png;base64,STUB',
    });
    expect('granted_by_person_id' in payload).toBe(false);
    expect(await screen.findByText('person detail')).toBeInTheDocument();
  });
});

describe('ConsentFormPage — errors, guardian and offline', () => {
  it('maps a 409 duplicate to the Portuguese message', async () => {
    createConsent.mockRejectedValue(new ApiError(409, { error: 'duplicate' }));
    renderPage();

    await sign();
    await submit();

    expect(
      await screen.findByText('Já existe um consentimento ativo deste tipo.'),
    ).toBeInTheDocument();
  });

  it('maps other API failures to a generic Portuguese message', async () => {
    createConsent.mockRejectedValue(new Error('boom'));
    renderPage();

    await sign();
    await submit();

    expect(
      await screen.findByText('Erro ao registrar consentimento. Tente novamente.'),
    ).toBeInTheDocument();
  });

  it('includes the guardian when selected for MINOR_GUARDIAN', async () => {
    createConsent.mockResolvedValue({ id: 'c-1' });
    renderPage();

    await userEvent.selectOptions(
      screen.getByLabelText('Tipo de consentimento'),
      'Responsável por menor',
    );
    await userEvent.click(screen.getByPlaceholderText('Buscar...'));
    await userEvent.click(screen.getByRole('option', { name: 'Maria Responsável' }));
    await sign();
    await submit();

    await waitFor(() =>
      expect(createConsent).toHaveBeenCalledWith(
        expect.objectContaining({
          consent_type: 'MINOR_GUARDIAN',
          granted_by_person_id: 'g-1',
        }),
      ),
    );
  });

  it('disables the form when offline', () => {
    offlineState.isOffline = true;
    renderPage();

    expect(
      screen.getByText('Conecte-se para registrar consentimentos.'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Tipo de consentimento')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Salvar' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'stub-assinar' })).toBeDisabled();
  });
});
