import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { TriageListItem, Pagination } from '../../types';

const useTriagesMock = vi.fn();
vi.mock('../../hooks/useTriages', () => ({
  useTriages: (...args: unknown[]) => useTriagesMock(...args),
}));

vi.mock('../../hooks/useCampusTimezone', () => ({
  useCampusTimezone: () => 'America/Sao_Paulo',
}));

import { TriageListPage } from '../TriageListPage';

const emptyPagination: Pagination = { page: 1, per_page: 20, total: 0, total_pages: 0 };

function mockTriages(overrides: Partial<ReturnType<typeof useTriagesMock>> = {}) {
  useTriagesMock.mockReturnValue({
    triages: [] as TriageListItem[],
    pagination: emptyPagination,
    isLoading: false,
    error: null,
    goToPage: vi.fn(),
    filterByPerson: vi.fn(),
    ...overrides,
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/triages']}>
      <Routes>
        <Route path="/triages" element={<TriageListPage />} />
        <Route path="/triages/new" element={<div>create page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('TriageListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('offers a control to start a new triage', () => {
    mockTriages();
    renderPage();
    expect(screen.getByRole('button', { name: /nova triagem/i })).toBeInTheDocument();
  });

  it('navigates to the triage creation flow when the control is clicked', async () => {
    mockTriages();
    renderPage();
    await userEvent.click(screen.getByRole('button', { name: /nova triagem/i }));
    expect(screen.getByText('create page')).toBeInTheDocument();
  });

  it('keeps the new-triage control available even when the list is not empty', () => {
    const triage = {
      id: 't1',
      person_id: 'p1',
      person_name: 'Maria Silva',
      main_complaint: 'Dor de cabeça',
      requested_service_count: 2,
      triage_date: '2026-07-07T12:00:00Z',
    } as unknown as TriageListItem;
    mockTriages({
      triages: [triage],
      pagination: { ...emptyPagination, total: 1, total_pages: 1 },
    });
    renderPage();
    expect(screen.getByRole('button', { name: /nova triagem/i })).toBeInTheDocument();
  });
});
