import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { DashboardMetrics } from '../../types';
import '../../components/charts/test-helpers';

const dashboardState: {
  data: DashboardMetrics | null;
  isLoading: boolean;
  error: string | null;
} = {
  data: null,
  isLoading: false,
  error: null,
};

vi.mock('../../hooks/useDashboard', () => ({
  useDashboard: () => dashboardState,
}));

vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({ email: 'coord@chesed.org', roles: ['COORDINATOR'] }),
}));

import { DashboardPage } from '../DashboardPage';

const METRICS: DashboardMetrics = {
  total_persons: 42,
  attendances_this_month: 8,
  upcoming_scheduled: 3,
  active_campaigns: 2,
  attendances_by_status: { COMPLETED: 30, SCHEDULED: 5 },
  recent_months: [
    { month: '2026-06', count: 9 },
    { month: '2026-07', count: 8 },
  ],
};

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );
}

describe('DashboardPage', () => {
  beforeEach(() => {
    dashboardState.data = null;
    dashboardState.isLoading = false;
    dashboardState.error = null;
  });

  it('renders KPI cards from the dashboard metrics', () => {
    dashboardState.data = METRICS;
    renderPage();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText(/pessoas cadastradas/i)).toBeInTheDocument();
    expect(screen.getByText(/atendimentos no mês/i)).toBeInTheDocument();
    expect(screen.getByText(/campanhas ativas/i)).toBeInTheDocument();
  });

  it('renders the trend and status charts when data is present', () => {
    dashboardState.data = METRICS;
    renderPage();
    expect(screen.getByTestId('monthly-trend-chart')).toBeInTheDocument();
    expect(screen.getByTestId('status-donut-chart')).toBeInTheDocument();
  });

  it('shows a loading placeholder while metrics load', () => {
    dashboardState.isLoading = true;
    renderPage();
    expect(screen.getAllByText('...').length).toBeGreaterThan(0);
  });

  it('surfaces the hook error', () => {
    dashboardState.error = 'Falha ao carregar painel';
    renderPage();
    expect(screen.getByText(/falha ao carregar painel/i)).toBeInTheDocument();
  });
});
