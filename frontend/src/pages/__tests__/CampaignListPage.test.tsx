import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { CampaignListItem, Pagination } from '../../types';

const hookState: {
  campaigns: CampaignListItem[];
  pagination: Pagination;
  isLoading: boolean;
  error: string | null;
} = {
  campaigns: [],
  pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
  isLoading: false,
  error: null,
};

vi.mock('../../hooks/useCampaigns', () => ({
  useCampaigns: () => ({
    ...hookState,
    goToPage: vi.fn(),
    filterByStatus: vi.fn(),
  }),
}));

const authState = { minRole: false };
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({ hasMinRole: () => authState.minRole }),
}));

import { CampaignListPage } from '../CampaignListPage';

function campaign(name: string): CampaignListItem {
  return {
    id: '00000000-0000-0000-0000-000000000001',
    name,
    campaign_type: 'SOCIAL_ACTION',
    status: 'ACTIVE',
    start_date: '2026-07-10T00:00:00Z',
    location_name: 'Community Center',
  };
}

function renderPage() {
  return render(
    <MemoryRouter>
      <CampaignListPage />
    </MemoryRouter>,
  );
}

describe('CampaignListPage', () => {
  beforeEach(() => {
    hookState.campaigns = [];
    hookState.isLoading = false;
    hookState.error = null;
    authState.minRole = false;
  });

  it('shows an empty state when there are no campaigns', () => {
    renderPage();
    expect(screen.getByText(/nenhuma campanha/i)).toBeInTheDocument();
  });

  it('renders campaign rows with translated type and status', () => {
    hookState.campaigns = [campaign('Ação de Julho')];
    renderPage();
    expect(screen.getByText('Ação de Julho')).toBeInTheDocument();
    expect(screen.getByText(/ação social/i)).toBeInTheDocument();
    expect(screen.getByText(/ativa/i)).toBeInTheDocument();
  });

  it('hides the create button for non-coordinators', () => {
    renderPage();
    expect(screen.queryByRole('link', { name: /nova campanha/i })).not.toBeInTheDocument();
  });

  it('shows the create button for coordinators', () => {
    authState.minRole = true;
    renderPage();
    expect(screen.getByRole('link', { name: /nova campanha/i })).toBeInTheDocument();
  });

  it('surfaces the hook error', () => {
    hookState.error = 'Falha ao carregar campanhas';
    renderPage();
    expect(screen.getByText(/falha ao carregar campanhas/i)).toBeInTheDocument();
  });
});
