import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { Consent } from '../../../types/consent';

const authState = { canView: true, isAdmin: false };
vi.mock('../../../hooks/useAuth', () => ({
  useAuth: () => ({
    hasMinRole: () => authState.canView,
    hasRole: () => authState.isAdmin,
  }),
}));

const offlineState = { isOffline: false };
vi.mock('../../../hooks/useOfflineStatus', () => ({
  useOfflineStatus: () => offlineState,
}));

const listPersonConsents = vi.fn();
const revokeConsent = vi.fn();
vi.mock('../../../api/consents', () => ({
  listPersonConsents: (...args: unknown[]) =>
    listPersonConsents(...args) as Promise<{ data: Consent[] }>,
  revokeConsent: (...args: unknown[]) => revokeConsent(...args) as Promise<Consent>,
}));

import { PersonConsentsSection } from '../PersonConsentsSection';

function consent(overrides: Partial<Consent> = {}): Consent {
  return {
    id: 'c-1',
    person_id: 'p-1',
    consent_type: 'DATA_PROCESSING',
    consent_version: '1.0',
    purpose: 'Cadastro e acompanhamento de atendimentos',
    granted_at: '2026-07-01T10:00:00Z',
    granted_by_person_id: null,
    signature_data: null,
    is_active: true,
    revoked_at: null,
    revoked_reason: null,
    ...overrides,
  };
}

function renderSection() {
  return render(
    <MemoryRouter>
      <PersonConsentsSection personId="p-1" />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  authState.canView = true;
  authState.isAdmin = false;
  offlineState.isOffline = false;
  listPersonConsents.mockReset();
  revokeConsent.mockReset();
  listPersonConsents.mockResolvedValue({ data: [consent()] });
});

describe('PersonConsentsSection — visibility and listing', () => {
  it('renders nothing for roles below SECRETARY', () => {
    authState.canView = false;
    const { container } = renderSection();

    expect(container.firstChild).toBeNull();
    expect(listPersonConsents).not.toHaveBeenCalled();
  });

  it('lists consents with a link to the new consent form for SECRETARY+', async () => {
    renderSection();

    expect(await screen.findByText('Tratamento de dados')).toBeInTheDocument();
    expect(screen.getByText('Consentimentos')).toBeInTheDocument();
    const newLink = screen.getByRole('link', { name: 'Novo consentimento' });
    expect(newLink).toHaveAttribute('href', '/persons/p-1/consents/new');
    expect(screen.queryByRole('button', { name: 'Revogar' })).not.toBeInTheDocument();
    expect(listPersonConsents).toHaveBeenCalledWith('p-1');
  });

  it('shows an offline message instead of the list when offline', async () => {
    offlineState.isOffline = true;
    renderSection();

    expect(
      await screen.findByText('Conecte-se para ver os consentimentos.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Tratamento de dados')).not.toBeInTheDocument();
    expect(listPersonConsents).not.toHaveBeenCalled();
  });

  it('surfaces a load failure as a readable message', async () => {
    listPersonConsents.mockRejectedValue(new Error('boom'));
    renderSection();

    expect(await screen.findByText('Erro ao carregar consentimentos')).toBeInTheDocument();
  });
});

describe('PersonConsentsSection — ADMIN revoke flow', () => {
  it('requires a reason before revoking', async () => {
    authState.isAdmin = true;
    renderSection();

    await userEvent.click(await screen.findByRole('button', { name: 'Revogar' }));
    await userEvent.click(screen.getByRole('button', { name: 'Confirmar revogação' }));

    expect(await screen.findByText('Informe o motivo da revogação.')).toBeInTheDocument();
    expect(revokeConsent).not.toHaveBeenCalled();
  });

  it('revokes with the reason and refreshes the list', async () => {
    authState.isAdmin = true;
    revokeConsent.mockResolvedValue(consent({ is_active: false }));
    renderSection();

    await userEvent.click(await screen.findByRole('button', { name: 'Revogar' }));
    await userEvent.type(
      screen.getByLabelText('Motivo da revogação'),
      'Pedido do titular',
    );
    await userEvent.click(screen.getByRole('button', { name: 'Confirmar revogação' }));

    await waitFor(() => expect(revokeConsent).toHaveBeenCalledWith('c-1', 'Pedido do titular'));
    await waitFor(() => expect(listPersonConsents).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Confirmar revogação' })).not.toBeInTheDocument(),
    );
  });
});
