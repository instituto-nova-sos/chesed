import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { AttendanceReport } from '../../types';
import '../../components/charts/test-helpers';

const reportState: {
  report: AttendanceReport | null;
  isLoading: boolean;
  error: string | null;
  generate: ReturnType<typeof vi.fn>;
} = {
  report: null,
  isLoading: false,
  error: null,
  generate: vi.fn(),
};

vi.mock('../../hooks/useAttendanceReport', () => ({
  useAttendanceReport: () => ({ ...reportState }),
}));

vi.mock('../../api/serviceTypes', () => ({
  listServiceTypes: () =>
    Promise.resolve({
      data: [{ id: 'st-1', code: 'LEGAL', name: 'Jurídico', is_active: true }],
    }),
}));

vi.mock('../../hooks/useCampaigns', () => ({
  useLinkableCampaigns: () => [],
}));

import { ReportsPage } from '../ReportsPage';

const REPORT: AttendanceReport = {
  period: { start: '2026-07-01', end: '2026-07-31' },
  total_attendances: 40,
  unique_persons: 33,
  by_status: { COMPLETED: 30, SCHEDULED: 5, IN_PROGRESS: 4, CANCELLED: 1 },
  by_service_type: [{ service_type: 'LEGAL', count: 12 }],
  by_month: [{ month: '2026-07', count: 40 }],
  by_professional: [
    {
      professional_id: 'a1',
      professional_name: 'Maria Silva',
      count: 25,
    },
  ],
};

function renderPage() {
  return render(
    <MemoryRouter>
      <ReportsPage />
    </MemoryRouter>,
  );
}

describe('ReportsPage', () => {
  beforeEach(() => {
    reportState.report = null;
    reportState.isLoading = false;
    reportState.error = null;
    reportState.generate = vi.fn();
  });

  it('renders the date-range filter and export button', () => {
    renderPage();
    expect(screen.getByRole('button', { name: /gerar relatório/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /exportar csv/i })).toBeInTheDocument();
  });

  it('renders the charts and professional breakdown when a report is present', () => {
    reportState.report = REPORT;
    renderPage();
    expect(screen.getByTestId('monthly-trend-chart')).toBeInTheDocument();
    expect(screen.getByTestId('status-donut-chart')).toBeInTheDocument();
    expect(screen.getByTestId('service-type-bar-chart')).toBeInTheDocument();
    const professionalCard = screen
      .getByRole('heading', { name: /por profissional/i })
      .closest('section') as HTMLElement;
    expect(within(professionalCard).getByText('Maria Silva')).toBeInTheDocument();
  });

  it('surfaces the hook error', () => {
    reportState.error = 'Falha ao gerar relatório';
    renderPage();
    expect(screen.getByText(/falha ao gerar relatório/i)).toBeInTheDocument();
  });
});
