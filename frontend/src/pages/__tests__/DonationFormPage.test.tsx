import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { DonationDetail, DonationInput } from '../../types';

const createDonation = vi.fn();
const updateDonation = vi.fn();
const getDonation = vi.fn();

vi.mock('../../api/donations', () => ({
  createDonation: (...args: unknown[]) => createDonation(...args) as Promise<DonationDetail>,
  updateDonation: (...args: unknown[]) => updateDonation(...args) as Promise<DonationDetail>,
  getDonation: (...args: unknown[]) => getDonation(...args) as Promise<DonationDetail>,
}));

vi.mock('../../hooks/useCampaigns', () => ({
  useLinkableCampaigns: () => [],
}));

vi.mock('../../hooks/usePersons', () => ({
  usePersons: () => ({ persons: [], search: vi.fn() }),
}));

import { DonationFormPage } from '../DonationFormPage';

function renderCreate() {
  return render(
    <MemoryRouter initialEntries={['/donations/new']}>
      <Routes>
        <Route path="/donations/new" element={<DonationFormPage />} />
        <Route path="/donations/:id" element={<div>detail page</div>} />
        <Route path="/donations" element={<div>list page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('DonationFormPage currency selector', () => {
  beforeEach(() => {
    createDonation.mockReset();
    updateDonation.mockReset();
    getDonation.mockReset();
  });

  it('renders a currency select with BRL, USD and EUR options defaulting to BRL', () => {
    renderCreate();
    const select = screen.getByLabelText(/moeda/i) as HTMLSelectElement;
    expect(select.tagName).toBe('SELECT');
    const values = Array.from(select.options).map((o) => o.value);
    expect(values).toEqual(expect.arrayContaining(['BRL', 'USD', 'EUR']));
    expect(select.value).toBe('BRL');
  });

  it('submits the chosen currency', async () => {
    createDonation.mockResolvedValue({ id: 'don-1' });
    renderCreate();

    await userEvent.type(screen.getByLabelText(/valor/i), '150');
    await userEvent.selectOptions(screen.getByLabelText(/moeda/i), 'USD');
    await userEvent.click(screen.getByRole('button', { name: /salvar/i }));

    await waitFor(() =>
      expect(createDonation).toHaveBeenCalledWith(
        expect.objectContaining({ currency: 'USD', amount: 150 } satisfies Partial<DonationInput>),
      ),
    );
  });
});
