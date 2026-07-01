import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
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

// The conflict badge links to /sync/conflicts, so the banner needs a router.
function renderBanner() {
  return render(
    <MemoryRouter>
      <SyncStatusBanner />
    </MemoryRouter>,
  );
}

describe('SyncStatusBanner', () => {
  beforeEach(() => {
    state.value = defaultState();
  });

  it('renders nothing when fully synced and online', () => {
    const { container } = renderBanner();
    expect(container).toBeEmptyDOMElement();
  });

  it('shows the pending count and a Sync Now button', () => {
    state.value = { ...defaultState(), pendingCount: 3 };
    renderBanner();
    expect(screen.getByText(/3/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /sincronizar/i })).toBeInTheDocument();
  });

  it('calls syncNow when the button is clicked', async () => {
    const syncNow = vi.fn();
    state.value = { ...defaultState(), pendingCount: 2, syncNow };
    renderBanner();
    await userEvent.click(screen.getByRole('button', { name: /sincronizar/i }));
    expect(syncNow).toHaveBeenCalledOnce();
  });

  it('surfaces a conflict warning linking to the resolution page', () => {
    state.value = { ...defaultState(), conflictCount: 1 };
    renderBanner();
    const link = screen.getByRole('link', { name: /conflito/i });
    expect(link).toHaveAttribute('href', '/sync/conflicts');
  });

  it('disables the button while syncing', () => {
    state.value = { ...defaultState(), pendingCount: 1, isSyncing: true };
    renderBanner();
    expect(screen.getByRole('button', { name: /sincroniz/i })).toBeDisabled();
  });

  // Regression guard: the offline notice was introduced in Track A and must
  // survive the S05.5 refactor (it is criterion 1, already satisfied).
  it('shows the offline state when the browser is offline', () => {
    state.value = { ...defaultState(), isOnline: false };
    renderBanner();
    expect(screen.getByText(/offline/i)).toBeInTheDocument();
  });

  it('shows a syncing indicator during an active drain even with zero pending', () => {
    // Mid-drain the queue can momentarily read 0 pending while records are
    // in flight; the banner must still surface the syncing state (S05.5).
    state.value = { ...defaultState(), isSyncing: true, pendingCount: 0 };
    renderBanner();
    expect(screen.getByText(/sincronizando/i)).toBeInTheDocument();
  });

  it('disables Sync Now and indicates it will run once online when offline', () => {
    state.value = { ...defaultState(), isOnline: false, pendingCount: 2 };
    renderBanner();
    const button = screen.getByRole('button', { name: /sincroniz/i });
    expect(button).toBeDisabled();
    // The user is told the action defers until connectivity returns.
    expect(button).toHaveAttribute('title', expect.stringMatching(/conex|online/i));
  });
});
