import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { CampaignListItem, PersonDetail } from '../../types';

const createAttendanceWithOfflineFallback = vi.fn();
vi.mock('../../offline/attendanceOffline', () => ({
  createAttendanceWithOfflineFallback: (...args: unknown[]) =>
    createAttendanceWithOfflineFallback(...args) as Promise<{ id: string; offline: boolean }>,
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

vi.mock('../../store/authStore', () => ({
  useAuthStore: (selector: (s: { personId: string | null }) => unknown) =>
    selector({ personId: 'prof-1' }),
}));

import { AttendanceCreatePage } from '../AttendanceCreatePage';

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
    <MemoryRouter initialEntries={['/attendances/new?person_id=person-1']}>
      <Routes>
        <Route path="/attendances/new" element={<AttendanceCreatePage />} />
        <Route path="/attendances" element={<div>list page</div>} />
        <Route path="/attendances/:id" element={<div>detail page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

async function fillRequiredAndSubmit() {
  await userEvent.selectOptions(screen.getByLabelText(/tipo de serviço/i), 'svc-1');
  await userEvent.click(screen.getByRole('button', { name: /agendar atendimento/i }));
  await waitFor(() => expect(createAttendanceWithOfflineFallback).toHaveBeenCalledOnce());
  return createAttendanceWithOfflineFallback.mock.calls[0]?.[0] as Record<string, unknown>;
}

describe('AttendanceCreatePage — campaign link', () => {
  beforeEach(() => {
    createAttendanceWithOfflineFallback.mockReset();
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
        campaign('camp-done', 'Campanha Concluída', 'COMPLETED'),
      ]),
    );
    createAttendanceWithOfflineFallback.mockResolvedValue({ id: 'a-1', offline: false });
  });

  it('renders the optional selector with only linkable campaigns', async () => {
    renderPage();
    expect(await screen.findByLabelText(/campanha \(opcional\)/i)).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Campanha Ativa' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'Sem campanha' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Campanha Concluída' })).not.toBeInTheDocument();
  });

  it('includes campaign_id in the create input when a campaign is selected', async () => {
    renderPage();
    const select = await screen.findByLabelText(/campanha \(opcional\)/i);
    await screen.findByRole('option', { name: 'Campanha Ativa' });
    await userEvent.selectOptions(select, 'camp-active');

    const input = await fillRequiredAndSubmit();
    expect(input.campaign_id).toBe('camp-active');
    expect(input.service_type_id).toBe('svc-1');
    expect(input.professional_id).toBe('prof-1');
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

    const input = await fillRequiredAndSubmit();
    expect('campaign_id' in input).toBe(false);
  });
});
