import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { CampaignDetail, CampaignMetrics } from '../../types';

const removeMember = vi.fn();
const addMember = vi.fn();
const detailState: {
  campaign: CampaignDetail | null;
  isLoading: boolean;
  error: string | null;
} = { campaign: null, isLoading: false, error: null };

vi.mock('../../hooks/useCampaign', () => ({
  useCampaign: () => ({
    ...detailState,
    addMember,
    removeMember,
    reload: vi.fn(),
  }),
}));

const metricsState: {
  metrics: CampaignMetrics | null;
  isLoading: boolean;
  error: string | null;
} = { metrics: null, isLoading: false, error: null };

vi.mock('../../hooks/useCampaignMetrics', () => ({
  useCampaignMetrics: () => metricsState,
}));

const authState = { minRole: false };
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({ hasMinRole: () => authState.minRole }),
}));

const personsSearch = vi.fn();
vi.mock('../../hooks/usePersons', () => ({
  usePersons: () => ({
    persons: [],
    isLoading: false,
    error: null,
    search: personsSearch,
    pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
  }),
}));

import { CampaignDetailPage } from '../CampaignDetailPage';

function detail(): CampaignDetail {
  return {
    id: '00000000-0000-0000-0000-00000000c001',
    name: 'Ação de Julho',
    campaign_type: 'HEALTH',
    status: 'ACTIVE',
    start_date: '2026-07-10T00:00:00Z',
    campus_id: '00000000-0000-0000-0000-000000000010',
    created_at: '2026-07-01T10:00:00Z',
    updated_at: '2026-07-01T10:00:00Z',
    team: [
      {
        person_id: '00000000-0000-0000-0000-000000000002',
        person_name: 'Maria Silva',
        role_in_campaign: 'VOLUNTEER',
        assigned_at: '2026-07-01T11:00:00Z',
      },
    ],
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/campaigns/00000000-0000-0000-0000-00000000c001']}>
      <Routes>
        <Route path="/campaigns/:id" element={<CampaignDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('CampaignDetailPage', () => {
  beforeEach(() => {
    detailState.campaign = detail();
    detailState.isLoading = false;
    detailState.error = null;
    metricsState.metrics = null;
    metricsState.isLoading = false;
    metricsState.error = null;
    authState.minRole = false;
    removeMember.mockReset();
    addMember.mockReset();
  });

  it('renders the campaign name, translated status, and team members', () => {
    renderPage();
    expect(screen.getByText('Ação de Julho')).toBeInTheDocument();
    expect(screen.getByText(/ativa/i)).toBeInTheDocument();
    expect(screen.getByText('Maria Silva')).toBeInTheDocument();
    expect(screen.getByText(/voluntári/i)).toBeInTheDocument();
  });

  it('hides team management controls from non-coordinators', () => {
    renderPage();
    expect(screen.queryByRole('button', { name: /remover/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /adicionar/i })).not.toBeInTheDocument();
  });

  it('lets a coordinator remove a team member', async () => {
    authState.minRole = true;
    removeMember.mockResolvedValue(undefined);
    renderPage();

    await userEvent.click(screen.getByRole('button', { name: /remover/i }));
    expect(removeMember).toHaveBeenCalledWith('00000000-0000-0000-0000-000000000002');
  });

  it('renders the metrics section for coordinators', () => {
    authState.minRole = true;
    metricsState.metrics = {
      campaign_id: '00000000-0000-0000-0000-00000000c001',
      campaign_name: 'Ação de Julho',
      status: 'ACTIVE',
      period: { start_date: '2026-07-10T00:00:00Z' },
      triage_count: 42,
      attendance_total: 61,
      attendances_by_status: { COMPLETED: 50, SCHEDULED: 11 },
      team_size: 12,
    };
    renderPage();

    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('61')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
  });

  it('hides the metrics section from non-coordinators', () => {
    metricsState.metrics = {
      campaign_id: '00000000-0000-0000-0000-00000000c001',
      campaign_name: 'Ação de Julho',
      status: 'ACTIVE',
      period: { start_date: '2026-07-10T00:00:00Z' },
      triage_count: 42,
      attendance_total: 61,
      attendances_by_status: {},
      team_size: 12,
    };
    renderPage();
    expect(screen.queryByText(/indicadores/i)).not.toBeInTheDocument();
  });
});

describe('CampaignDetailPage — review remediation', () => {
  beforeEach(() => {
    detailState.campaign = detail();
    detailState.isLoading = false;
    detailState.error = null;
    authState.minRole = true;
    personsSearch.mockReset();
  });

  it('wires the team picker to the server-side person search (M3)', async () => {
    renderPage();
    const pickerInput = screen.getByPlaceholderText(/buscar/i);
    await userEvent.type(pickerInput, 'Mar');
    expect(personsSearch).toHaveBeenCalledWith('Mar');
  });
});
