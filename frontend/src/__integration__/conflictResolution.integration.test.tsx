/**
 * Integration test: S12.2 conflict-resolution UI end to end.
 *
 * Exercises the real chain — SyncConflictsPage → useSyncConflicts → syncEngine →
 * Dexie — with no module mocking, backed by fake-indexeddb. A conflicted record
 * and its captured server snapshot are seeded into IndexedDB; the page renders the
 * field-level diff, the operator picks "Manter servidor", and we assert the entity
 * cache converges to the server value ('synced') and the conflict is cleared.
 *
 * The keep-server resolution is local-only (no push), so no MSW handler is needed;
 * `./setup` still boots the shared server so any accidental fetch fails the test.
 */

import 'fake-indexeddb/auto';
import { beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';

import { db } from '../offline/db';
import { SyncConflictsPage } from '../pages/SyncConflictsPage';
import './setup';

const ENTITY_ID = '33333333-3333-3333-3333-333333333333';

async function seedConflictedPerson(): Promise<number> {
  const now = new Date().toISOString();
  await db.persons.put({
    id: ENTITY_ID,
    data: { id: ENTITY_ID, full_name: 'Local Name', phone: '111' },
    syncStatus: 'conflict',
    localCreatedAt: now,
  });
  await db.conflicts.put({
    entityId: ENTITY_ID,
    entityType: 'person',
    serverData: { id: ENTITY_ID, full_name: 'Server Name', phone: '999' },
    serverUpdatedAt: '2026-05-01T00:00:00Z',
    conflictFields: ['full_name', 'phone'],
    capturedAt: now,
  });
  return (await db.syncQueue.add({
    entityType: 'person',
    entityId: ENTITY_ID,
    action: 'update',
    data: { id: ENTITY_ID, full_name: 'Local Name', phone: '111' },
    createdAt: now,
    retryCount: 1,
    conflicted: true,
    lastError: 'conflict',
  })) as number;
}

function renderPage() {
  return render(
    <MemoryRouter>
      <SyncConflictsPage />
    </MemoryRouter>,
  );
}

describe('integration: conflict resolution UI + syncEngine + Dexie', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
  });

  it('shows the local vs server diff for a captured conflict', async () => {
    await seedConflictedPerson();
    renderPage();

    // Wait for the diff to mount (its keep-server action is diff-only) before
    // asserting the field values, so we don't race the fallback list render.
    await screen.findByRole('button', { name: /manter servidor/i });
    expect(screen.getByText('Local Name')).toBeInTheDocument();
    expect(screen.getByText('Server Name')).toBeInTheDocument();
    expect(screen.getByText('111')).toBeInTheDocument();
    expect(screen.getByText('999')).toBeInTheDocument();
  });

  it('applies the server version and clears the conflict on "Manter servidor"', async () => {
    await seedConflictedPerson();
    renderPage();

    await userEvent.click(
      await screen.findByRole('button', { name: /manter servidor/i }),
    );

    await waitFor(async () => {
      expect(await db.conflicts.get(ENTITY_ID)).toBeUndefined();
    });

    const person = await db.persons.get(ENTITY_ID);
    expect(person?.syncStatus).toBe('synced');
    expect(person?.data.full_name).toBe('Server Name');
    expect(person?.data.phone).toBe('999');
    expect(await db.syncQueue.count()).toBe(0);

    await waitFor(() =>
      expect(screen.getByText(/nenhum conflito/i)).toBeInTheDocument(),
    );
  });

  it('re-pushes a merged payload and marks the entity pending on "Aplicar mesclagem"', async () => {
    const queueId = await seedConflictedPerson();
    renderPage();

    // Default conflict-field selection is "server"; flip full_name to local so the
    // merged record mixes both sources (local name + server phone).
    await userEvent.click(
      await screen.findByRole('button', { name: /full_name.*usar local/i }),
    );
    await userEvent.click(
      screen.getByRole('button', { name: /aplicar mesclagem/i }),
    );

    await waitFor(async () => {
      expect(await db.conflicts.get(ENTITY_ID)).toBeUndefined();
    });

    const queued = await db.syncQueue.get(queueId);
    expect(queued?.conflicted).toBeFalsy();
    expect(queued?.data.full_name).toBe('Local Name');
    expect(queued?.data.phone).toBe('999');

    const person = await db.persons.get(ENTITY_ID);
    expect(person?.syncStatus).toBe('pending');
    expect(person?.data.full_name).toBe('Local Name');
  });
});
