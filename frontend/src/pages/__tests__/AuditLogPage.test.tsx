import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { AuditLogPage as AuditLogPageData } from '../../types/audit';

const offlineState = { isOffline: false };
vi.mock('../../hooks/useOfflineStatus', () => ({
  useOfflineStatus: () => offlineState,
}));

const listAuditLogs = vi.fn();
vi.mock('../../api/audit', () => ({
  listAuditLogs: (...args: unknown[]) =>
    listAuditLogs(...args) as Promise<AuditLogPageData>,
}));

import { AuditLogPage } from '../AuditLogPage';

function page(overrides: Partial<AuditLogPageData> = {}): AuditLogPageData {
  return {
    data: [
      {
        id: 'a-1',
        user_email: 'maria@example.com',
        action_type: 'UPDATE',
        entity_type: 'person',
        entity_id: 'p-1',
        description: 'Updated phone number',
        old_values: null,
        new_values: null,
        ip_address: '192.168.1.1',
        timestamp: '2026-04-02T10:30:00Z',
      },
    ],
    pagination: { page: 1, per_page: 50, total: 1, total_pages: 1 },
    ...overrides,
  };
}

beforeEach(() => {
  offlineState.isOffline = false;
  listAuditLogs.mockReset();
});

describe('AuditLogPage', () => {
  it('renders audit rows returned by the API', async () => {
    listAuditLogs.mockResolvedValue(page());
    render(<AuditLogPage />);

    await waitFor(() =>
      expect(screen.getByText('maria@example.com')).toBeInTheDocument(),
    );
    expect(screen.getByText('person')).toBeInTheDocument();
    expect(screen.getByText('Updated phone number')).toBeInTheDocument();
  });

  it('re-queries with filters when the form is submitted', async () => {
    listAuditLogs.mockResolvedValue(page());
    const user = userEvent.setup();
    render(<AuditLogPage />);

    await waitFor(() => expect(listAuditLogs).toHaveBeenCalledTimes(1));

    const entity = screen.getByLabelText(/Entidade/i);
    await user.type(entity, 'person');
    await user.click(screen.getByRole('button', { name: /Buscar/i }));

    await waitFor(() => expect(listAuditLogs).toHaveBeenCalledTimes(2));
    const lastCall = listAuditLogs.mock.calls.at(-1)?.[0] as {
      entity_type?: string;
    };
    expect(lastCall.entity_type).toBe('person');
  });

  it('shows an offline message and does not fetch when offline', async () => {
    offlineState.isOffline = true;
    render(<AuditLogPage />);

    expect(screen.getByText(/offline/i)).toBeInTheDocument();
    expect(listAuditLogs).not.toHaveBeenCalled();
  });

  it('shows an empty state when there are no entries', async () => {
    listAuditLogs.mockResolvedValue(
      page({ data: [], pagination: { page: 1, per_page: 50, total: 0, total_pages: 0 } }),
    );
    render(<AuditLogPage />);

    await waitFor(() =>
      expect(screen.getByText(/Nenhum registro/i)).toBeInTheDocument(),
    );
  });
});
