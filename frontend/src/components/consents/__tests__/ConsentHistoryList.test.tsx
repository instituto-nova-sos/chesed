import { describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ConsentHistoryList } from '../ConsentHistoryList';
import type { Consent } from '../../../types/consent';

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

const ROWS: Consent[] = [
  consent(),
  consent({
    id: 'c-2',
    consent_type: 'IMAGE_USAGE',
    granted_at: '2026-05-02T09:00:00Z',
    is_active: false,
    revoked_at: '2026-07-01T10:00:00Z',
    revoked_reason: 'Pedido do titular',
  }),
];

describe('ConsentHistoryList', () => {
  it('renders type label, version, granted date, and status badges in Portuguese', () => {
    render(<ConsentHistoryList consents={ROWS} canRevoke={false} onRevoke={vi.fn()} />);

    expect(screen.getByText('Tratamento de dados')).toBeInTheDocument();
    expect(screen.getByText('Uso de imagem')).toBeInTheDocument();
    expect(screen.getAllByText(/1\.0/)).not.toHaveLength(0);
    expect(screen.getByText(/01\/07\/2026/)).toBeInTheDocument();
    expect(screen.getByText(/02\/05\/2026/)).toBeInTheDocument();
    expect(screen.getByText('Ativo')).toBeInTheDocument();
    expect(screen.getByText('Revogado')).toBeInTheDocument();
    expect(screen.getByText(/Pedido do titular/)).toBeInTheDocument();
  });

  it('preserves the order of the consents as returned by the API', () => {
    render(<ConsentHistoryList consents={ROWS} canRevoke={false} onRevoke={vi.fn()} />);

    const [first, second] = screen.getAllByRole('listitem');
    expect(first).toBeDefined();
    expect(second).toBeDefined();
    expect(within(first as HTMLElement).getByText('Tratamento de dados')).toBeInTheDocument();
    expect(within(second as HTMLElement).getByText('Uso de imagem')).toBeInTheDocument();
  });

  it('shows the revoke action only on active consents when allowed', async () => {
    const onRevoke = vi.fn();
    render(<ConsentHistoryList consents={ROWS} canRevoke onRevoke={onRevoke} />);

    const revokeButtons = screen.getAllByRole('button', { name: 'Revogar' });
    expect(revokeButtons).toHaveLength(1);

    await userEvent.click(revokeButtons[0] as HTMLElement);
    expect(onRevoke).toHaveBeenCalledWith(ROWS[0]);
  });

  it('hides the revoke action when not allowed', () => {
    render(<ConsentHistoryList consents={ROWS} canRevoke={false} onRevoke={vi.fn()} />);
    expect(screen.queryByRole('button', { name: 'Revogar' })).not.toBeInTheDocument();
  });

  it('renders an empty state when there are no consents', () => {
    render(<ConsentHistoryList consents={[]} canRevoke={false} onRevoke={vi.fn()} />);
    expect(screen.getByText('Nenhum consentimento registrado.')).toBeInTheDocument();
  });
});
