import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { Campaign } from '../../types';

const createCampaign = vi.fn();
const updateCampaign = vi.fn();
const getCampaign = vi.fn();

vi.mock('../../api/campaigns', () => ({
  createCampaign: (...args: unknown[]) => createCampaign(...args) as Promise<Campaign>,
  updateCampaign: (...args: unknown[]) => updateCampaign(...args) as Promise<Campaign>,
  getCampaign: (...args: unknown[]) => getCampaign(...args) as Promise<Campaign>,
}));

import { CampaignFormPage } from '../CampaignFormPage';

function renderEdit(id: string) {
  return render(
    <MemoryRouter initialEntries={[`/campaigns/${id}/edit`]}>
      <Routes>
        <Route path="/campaigns/:id/edit" element={<CampaignFormPage />} />
        <Route path="/campaigns/:id" element={<div>detail page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

function renderCreate() {
  return render(
    <MemoryRouter initialEntries={['/campaigns/new']}>
      <Routes>
        <Route path="/campaigns/new" element={<CampaignFormPage />} />
        <Route path="/campaigns/:id" element={<div>detail page</div>} />
        <Route path="/campaigns" element={<div>list page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('CampaignFormPage', () => {
  beforeEach(() => {
    createCampaign.mockReset();
    updateCampaign.mockReset();
    getCampaign.mockReset();
  });

  it('renders the required fields in Portuguese', () => {
    renderCreate();
    expect(screen.getByLabelText(/nome/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/tipo/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/data de início/i)).toBeInTheDocument();
  });

  it('submits the campaign and navigates away', async () => {
    createCampaign.mockResolvedValue({
      id: '00000000-0000-0000-0000-00000000c001',
      name: 'Nova Ação',
      campaign_type: 'SOCIAL_ACTION',
      status: 'PLANNED',
      start_date: '2026-07-10T00:00:00Z',
      campus_id: 'x',
      created_at: 'x',
      updated_at: 'x',
    });
    renderCreate();

    await userEvent.type(screen.getByLabelText(/nome/i), 'Nova Ação');
    await userEvent.type(screen.getByLabelText(/data de início/i), '2026-07-10');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    await waitFor(() =>
      expect(createCampaign).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Nova Ação', start_date: '2026-07-10' }),
      ),
    );
  });

  it('shows a required-field message when name is missing', async () => {
    renderCreate();
    await userEvent.type(screen.getByLabelText(/data de início/i), '2026-07-10');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    expect(await screen.findByText(/obrigatório/i)).toBeInTheDocument();
    expect(createCampaign).not.toHaveBeenCalled();
  });

  it('surfaces an API failure as a readable message', async () => {
    createCampaign.mockRejectedValue(new Error('boom'));
    renderCreate();

    await userEvent.type(screen.getByLabelText(/nome/i), 'Nova Ação');
    await userEvent.type(screen.getByLabelText(/data de início/i), '2026-07-10');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    expect(await screen.findByText(/erro ao salvar campanha/i)).toBeInTheDocument();
  });
});

describe('CampaignFormPage — review remediation', () => {
  beforeEach(() => {
    createCampaign.mockReset();
    updateCampaign.mockReset();
    getCampaign.mockReset();
  });

  it('preserves coordinator_id through an edit round-trip (M2)', async () => {
    const id = '00000000-0000-0000-0000-00000000c001';
    getCampaign.mockResolvedValue({
      id,
      name: 'Com Coordenador',
      campaign_type: 'HEALTH',
      status: 'ACTIVE',
      start_date: '2026-07-10T00:00:00Z',
      coordinator_id: '00000000-0000-0000-0000-000000000009',
      campus_id: 'x',
      created_at: 'x',
      updated_at: 'x',
    });
    updateCampaign.mockResolvedValue({ id });
    renderEdit(id);

    expect(await screen.findByDisplayValue('Com Coordenador')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    await waitFor(() =>
      expect(updateCampaign).toHaveBeenCalledWith(
        id,
        expect.objectContaining({
          coordinator_id: '00000000-0000-0000-0000-000000000009',
          status: 'ACTIVE',
        }),
      ),
    );
  });

  it('rejects an end date earlier than the start date client-side (N3)', async () => {
    renderCreate();
    await userEvent.type(screen.getByLabelText(/nome/i), 'Datas Invertidas');
    await userEvent.type(screen.getByLabelText(/data de início/i), '2026-07-10');
    await userEvent.type(screen.getByLabelText(/data de término/i), '2026-07-01');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    expect(await screen.findByText(/anterior/i)).toBeInTheDocument();
    expect(createCampaign).not.toHaveBeenCalled();
  });
});
