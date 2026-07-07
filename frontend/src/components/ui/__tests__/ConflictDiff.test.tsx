import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConflictDiff } from '../ConflictDiff';
import type { ConflictSnapshot } from '../../../offline/db';

function makeSnapshot(
  serverData: Record<string, unknown>,
  conflictFields?: string[],
): ConflictSnapshot {
  return {
    entityId: 'entity-1',
    entityType: 'person',
    serverData,
    serverUpdatedAt: '2026-05-01T00:00:00Z',
    conflictFields,
    capturedAt: '2026-05-01T01:00:00Z',
  };
}

describe('ConflictDiff', () => {
  const noop = () => undefined;

  it('renders each field with its local and server value', () => {
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name', phone: '111' }}
        snapshot={makeSnapshot({ full_name: 'Server Name', phone: '111' })}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={noop}
      />,
    );

    expect(screen.getByText('Local Name')).toBeInTheDocument();
    expect(screen.getByText('Server Name')).toBeInTheDocument();
    expect(screen.getByText('full_name')).toBeInTheDocument();
    expect(screen.getByText('phone')).toBeInTheDocument();
  });

  it('renders the union of local-only and server-only keys', () => {
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name', only_local: 'L' }}
        snapshot={makeSnapshot({ full_name: 'Server Name', only_server: 'S' })}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={noop}
      />,
    );

    expect(screen.getByText('only_local')).toBeInTheDocument();
    expect(screen.getByText('only_server')).toBeInTheDocument();
  });

  it('marks a conflicting field row so it is visually highlighted', () => {
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name', phone: '111' }}
        snapshot={makeSnapshot(
          { full_name: 'Server Name', phone: '111' },
          ['full_name'],
        )}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={noop}
      />,
    );

    expect(screen.getByTestId('conflict-field-full_name')).toBeInTheDocument();
    expect(screen.queryByTestId('conflict-field-phone')).not.toBeInTheDocument();
  });

  it('renders object values as compact JSON and missing values as a dash', () => {
    render(
      <ConflictDiff
        localData={{ meta: { a: 1 } }}
        snapshot={makeSnapshot({ meta: undefined })}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={noop}
      />,
    );

    expect(screen.getByText('{"a":1}')).toBeInTheDocument();
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('calls onKeepLocal when "Manter local" is clicked', async () => {
    const onKeepLocal = vi.fn();
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name' }}
        snapshot={makeSnapshot({ full_name: 'Server Name' }, ['full_name'])}
        onKeepLocal={onKeepLocal}
        onKeepServer={noop}
        onMerge={noop}
      />,
    );

    await userEvent.click(screen.getByRole('button', { name: /manter local/i }));
    expect(onKeepLocal).toHaveBeenCalledTimes(1);
  });

  it('calls onKeepServer when "Manter servidor" is clicked', async () => {
    const onKeepServer = vi.fn();
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name' }}
        snapshot={makeSnapshot({ full_name: 'Server Name' }, ['full_name'])}
        onKeepLocal={noop}
        onKeepServer={onKeepServer}
        onMerge={noop}
      />,
    );

    await userEvent.click(screen.getByRole('button', { name: /manter servidor/i }));
    expect(onKeepServer).toHaveBeenCalledTimes(1);
  });

  it('builds a merged payload from per-field selections and calls onMerge', async () => {
    const onMerge = vi.fn();
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name', phone: '111' }}
        snapshot={makeSnapshot(
          { full_name: 'Server Name', phone: '999' },
          ['full_name', 'phone'],
        )}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={onMerge}
      />,
    );

    // Conflicting fields default to "Servidor"; pick local for full_name, keep
    // server for phone, so the merged record mixes the two sources.
    await userEvent.click(
      screen.getByRole('button', { name: /full_name.*usar local/i }),
    );
    await userEvent.click(screen.getByRole('button', { name: /aplicar mesclagem/i }));

    expect(onMerge).toHaveBeenCalledTimes(1);
    expect(onMerge).toHaveBeenCalledWith({
      full_name: 'Local Name',
      phone: '999',
    });
  });

  it('defaults non-conflicting fields to the local value in the merge', async () => {
    const onMerge = vi.fn();
    render(
      <ConflictDiff
        localData={{ full_name: 'Local Name', phone: '111' }}
        snapshot={makeSnapshot(
          { full_name: 'Server Name', phone: '111' },
          ['full_name'],
        )}
        onKeepLocal={noop}
        onKeepServer={noop}
        onMerge={onMerge}
      />,
    );

    await userEvent.click(screen.getByRole('button', { name: /aplicar mesclagem/i }));

    expect(onMerge).toHaveBeenCalledWith({
      full_name: 'Server Name',
      phone: '111',
    });
  });
});
