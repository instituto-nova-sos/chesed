import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { SyncQueueItem } from '../../offline/db';

const discard = vi.fn();
const resubmit = vi.fn();
const state: { conflicts: SyncQueueItem[]; isLoading: boolean } = {
  conflicts: [],
  isLoading: false,
};

vi.mock('../../hooks/useSyncConflicts', () => ({
  useSyncConflicts: () => ({
    conflicts: state.conflicts,
    isLoading: state.isLoading,
    discard,
    resubmit,
  }),
}));

import { SyncConflictsPage } from '../SyncConflictsPage';

function conflict(id: string, entityType: SyncQueueItem['entityType']): SyncQueueItem {
  return {
    id: Number(id),
    entityType,
    entityId: `entity-${id}`,
    action: 'create',
    data: { full_name: `Record ${id}` },
    createdAt: '2026-06-30T00:00:00Z',
    retryCount: 1,
    conflicted: true,
    lastError: 'server conflict',
  };
}

function renderPage() {
  return render(
    <MemoryRouter>
      <SyncConflictsPage />
    </MemoryRouter>,
  );
}

describe('SyncConflictsPage', () => {
  beforeEach(() => {
    state.conflicts = [];
    state.isLoading = false;
    discard.mockReset();
    resubmit.mockReset();
  });

  it('shows an empty state when there are no conflicts', () => {
    renderPage();
    expect(screen.getByText(/nenhum conflito/i)).toBeInTheDocument();
  });

  it('lists each conflicted record with its entity type', () => {
    state.conflicts = [conflict('1', 'triage'), conflict('2', 'attendance')];
    renderPage();
    expect(screen.getByText(/Record 1/)).toBeInTheDocument();
    expect(screen.getByText(/Record 2/)).toBeInTheDocument();
    expect(screen.getByText(/triagem/i)).toBeInTheDocument();
    expect(screen.getByText(/atendimento/i)).toBeInTheDocument();
  });

  it('calls resubmit with the queue id when Resubmit is clicked', async () => {
    state.conflicts = [conflict('7', 'person')];
    renderPage();
    await userEvent.click(screen.getByRole('button', { name: /reenviar/i }));
    expect(resubmit).toHaveBeenCalledWith(7);
  });

  it('calls discard with the queue id when Discard is clicked', async () => {
    state.conflicts = [conflict('9', 'person')];
    renderPage();
    await userEvent.click(screen.getByRole('button', { name: /descartar/i }));
    await waitFor(() => expect(discard).toHaveBeenCalledWith(9));
  });
});
