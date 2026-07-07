import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { ConflictSnapshot, SyncQueueItem } from '../../offline/db';

const discard = vi.fn();
const resubmit = vi.fn();
const keepLocal = vi.fn();
const keepServer = vi.fn();
const merge = vi.fn();
const snapshotFor = vi.fn<(entityId: string) => Promise<ConflictSnapshot | undefined>>();

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
    keepLocal,
    keepServer,
    merge,
    snapshotFor,
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

function snapshotFrom(
  entityId: string,
  serverData: Record<string, unknown>,
  conflictFields?: string[],
): ConflictSnapshot {
  return {
    entityId,
    entityType: 'person',
    serverData,
    serverUpdatedAt: '2026-05-01T00:00:00Z',
    conflictFields,
    capturedAt: '2026-05-01T01:00:00Z',
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
    keepLocal.mockReset();
    keepServer.mockReset();
    merge.mockReset();
    snapshotFor.mockReset();
    snapshotFor.mockResolvedValue(undefined);
  });

  it('shows an empty state when there are no conflicts', () => {
    renderPage();
    expect(screen.getByText(/nenhum conflito/i)).toBeInTheDocument();
  });

  describe('without a captured snapshot (fallback path)', () => {
    it('lists each conflicted record with its entity type', async () => {
      state.conflicts = [conflict('1', 'triage'), conflict('2', 'attendance')];
      renderPage();
      await waitFor(() => expect(screen.getByText(/Record 1/)).toBeInTheDocument());
      expect(screen.getByText(/Record 2/)).toBeInTheDocument();
      expect(screen.getByText(/triagem/i)).toBeInTheDocument();
      expect(screen.getByText(/atendimento/i)).toBeInTheDocument();
    });

    it('calls resubmit with the queue id when Resubmit is clicked', async () => {
      state.conflicts = [conflict('7', 'person')];
      renderPage();
      await userEvent.click(await screen.findByRole('button', { name: /reenviar/i }));
      expect(resubmit).toHaveBeenCalledWith(7);
    });

    it('calls discard with the queue id when Discard is clicked', async () => {
      state.conflicts = [conflict('9', 'person')];
      renderPage();
      await userEvent.click(await screen.findByRole('button', { name: /descartar/i }));
      await waitFor(() => expect(discard).toHaveBeenCalledWith(9));
    });
  });

  describe('with a captured snapshot (diff path)', () => {
    it('renders the local and server values from the snapshot diff', async () => {
      state.conflicts = [conflict('5', 'person')];
      snapshotFor.mockResolvedValue(
        snapshotFrom('entity-5', { full_name: 'Server Name' }, ['full_name']),
      );
      renderPage();

      expect(await screen.findByText('Server Name')).toBeInTheDocument();
      expect(screen.getByText('Record 5')).toBeInTheDocument();
    });

    it('calls keepServer with the queue id when "Manter servidor" is clicked', async () => {
      state.conflicts = [conflict('5', 'person')];
      snapshotFor.mockResolvedValue(
        snapshotFrom('entity-5', { full_name: 'Server Name' }, ['full_name']),
      );
      renderPage();

      await userEvent.click(
        await screen.findByRole('button', { name: /manter servidor/i }),
      );
      expect(keepServer).toHaveBeenCalledWith(5);
    });

    it('calls merge with the queue id and merged payload when applied', async () => {
      state.conflicts = [conflict('5', 'person')];
      snapshotFor.mockResolvedValue(
        snapshotFrom(
          'entity-5',
          { full_name: 'Server Name' },
          ['full_name'],
        ),
      );
      renderPage();

      await userEvent.click(
        await screen.findByRole('button', { name: /aplicar mesclagem/i }),
      );
      expect(merge).toHaveBeenCalledTimes(1);
      expect(merge).toHaveBeenCalledWith(5, { full_name: 'Server Name' });
    });
  });
});
