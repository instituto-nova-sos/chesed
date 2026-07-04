import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { CampaignListItem, PersonDetail } from '../../types';

const createTriageWithOfflineFallback = vi.fn();
vi.mock('../../offline/triageOffline', () => ({
  createTriageWithOfflineFallback: (...args: unknown[]) =>
    createTriageWithOfflineFallback(...args) as Promise<{ id: string; offline: boolean }>,
}));

const getPerson = vi.fn();
vi.mock('../../api/persons', () => ({
  getPerson: (...args: unknown[]) => getPerson(...args) as Promise<PersonDetail>,
}));

const listServiceTypes = vi.fn();
vi.mock('../../api/serviceTypes', () => ({
  listServiceTypes: (...args: unknown[]) => listServiceTypes(...args),
}));

const listCampaigns = vi.fn();
vi.mock('../../api/campaigns', () => ({
  listCampaigns: (...args: unknown[]) => listCampaigns(...args),
}));

import { TriageCreatePage } from '../TriageCreatePage';

const person = { id: 'person-1', full_name: 'Maria Silva' } as unknown as PersonDetail;

function campaign(id: string, name: string, status: string): CampaignListItem {
  return { id, name, campaign_type: 'SOCIAL_ACTION', status, start_date: '2026-07-10T00:00:00Z' };
}

function campaignsResponse(data: CampaignListItem[]) {
  return {
    data,
    pagination: { page: 1, per_page: 100, total: data.length, total_pages: 1 },
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/triages/new?person_id=person-1']}>
      <Routes>
        <Route path="/triages/new" element={<TriageCreatePage />} />
        <Route path="/triages" element={<div>list page</div>} />
        <Route path="/triages/:id" element={<div>detail page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

async function fillRequiredAndSubmit() {
  await userEvent.type(screen.getByLabelText(/queixa principal/i), 'Dor de cabeça');
  await userEvent.click(screen.getByRole('button', { name: /salvar triagem/i }));
  await waitFor(() => expect(createTriageWithOfflineFallback).toHaveBeenCalledOnce());
  return createTriageWithOfflineFallback.mock.calls[0]?.[0] as Record<string, unknown>;
}

describe('TriageCreatePage — campaign link', () => {
  beforeEach(() => {
    createTriageWithOfflineFallback.mockReset();
    getPerson.mockReset();
    listServiceTypes.mockReset();
    listCampaigns.mockReset();
    getPerson.mockResolvedValue(person);
    listServiceTypes.mockResolvedValue({
      data: [{ id: 'svc-1', name: 'Consulta', is_active: true }],
    });
    listCampaigns.mockResolvedValue(
      campaignsResponse([
        campaign('camp-active', 'Campanha Ativa', 'ACTIVE'),
        campaign('camp-planned', 'Campanha Planejada', 'PLANNED'),
        campaign('camp-done', 'Campanha Concluída', 'COMPLETED'),
      ]),
    );
    createTriageWithOfflineFallback.mockResolvedValue({ id: 't-1', offline: false });
  });

  it('renders the optional selector with only linkable campaigns', async () => {
    renderPage();
    expect(await screen.findByLabelText(/campanha \(opcional\)/i)).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Campanha Ativa' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Sem campanha' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Campanha Planejada' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Campanha Concluída' })).not.toBeInTheDocument();
  });

  it('includes campaign_id in the create input when a campaign is selected', async () => {
    renderPage();
    const select = await screen.findByLabelText(/campanha \(opcional\)/i);
    await screen.findByRole('option', { name: 'Campanha Ativa' });
    await userEvent.selectOptions(select, 'camp-active');

    const input = await fillRequiredAndSubmit();
    expect(input.campaign_id).toBe('camp-active');
    expect(input.person_id).toBe('person-1');
  });

  it('omits the campaign_id key entirely when no campaign is selected', async () => {
    renderPage();
    await screen.findByLabelText(/campanha \(opcional\)/i);

    const input = await fillRequiredAndSubmit();
    expect('campaign_id' in input).toBe(false);
  });

  it('still submits when there are no campaigns, rendering only the empty option', async () => {
    listCampaigns.mockResolvedValue(campaignsResponse([]));
    renderPage();
    const select = await screen.findByLabelText(/campanha \(opcional\)/i);
    expect(select.querySelectorAll('option')).toHaveLength(1);
    expect(screen.getByRole('option', { name: 'Sem campanha' })).toBeInTheDocument();

    const input = await fillRequiredAndSubmit();
    expect('campaign_id' in input).toBe(false);
  });

  it('keeps the form usable when the campaigns request fails', async () => {
    listCampaigns.mockRejectedValue(new Error('campaigns down'));
    renderPage();
    const select = await screen.findByLabelText(/campanha \(opcional\)/i);
    expect(select.querySelectorAll('option')).toHaveLength(1);
    expect(screen.getByText('Maria Silva')).toBeInTheDocument();

    const input = await fillRequiredAndSubmit();
    expect('campaign_id' in input).toBe(false);
  });
});
