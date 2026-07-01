import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { OnlineSyncState } from '../../../hooks/useOnlineSync';

const state: { value: OnlineSyncState } = { value: defaultState() };
function defaultState(): OnlineSyncState {
  return {
    isOnline: true,
    isSyncing: false,
    pendingCount: 0,
    conflictCount: 0,
    lastError: null,
    syncNow: vi.fn(),
  };
}

vi.mock('../../../hooks/useOnlineSync', () => ({
  useOnlineSync: () => state.value,
}));

import { SyncStatusBanner } from '../SyncStatusBanner';

describe('SyncStatusBanner', () => {
  beforeEach(() => {
    state.value = defaultState();
  });

  it('renders nothing when fully synced and online', () => {
    const { container } = render(<SyncStatusBanner />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows the pending count and a Sync Now button', () => {
    state.value = { ...defaultState(), pendingCount: 3 };
    render(<SyncStatusBanner />);
    expect(screen.getByText(/3/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sincronizar/i })).toBeInTheDocument();
  });

  it('calls syncNow when the button is clicked', async () => {
    const syncNow = vi.fn();
    state.value = { ...defaultState(), pendingCount: 2, syncNow };
    render(<SyncStatusBanner />);
    await userEvent.click(screen.getByRole('button', { name: /sincronizar/i }));
    expect(syncNow).toHaveBeenCalledOnce();
  });

  it('surfaces a conflict warning when conflicts exist', () => {
    state.value = { ...defaultState(), conflictCount: 1 };
    render(<SyncStatusBanner />);
    expect(screen.getByText(/conflito/i)).toBeInTheDocument();
  });

  it('disables the button while syncing', () => {
    state.value = { ...defaultState(), pendingCount: 1, isSyncing: true };
    render(<SyncStatusBanner />);
    expect(screen.getByRole('button', { name: /sincroniz/i })).toBeDisabled();
  });
});
