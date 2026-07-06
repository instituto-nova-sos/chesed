import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import type { DonationDetail } from '../../types';

const donationState: {
  donation: DonationDetail | null;
  isLoading: boolean;
  error: string | null;
} = { donation: null, isLoading: false, error: null };

vi.mock('../../hooks/useDonation', () => ({
  useDonation: () => ({ ...donationState }),
}));

const roleState = { min: 'COORDINATOR' };
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    hasMinRole: (role: string) => {
      const order = ['VOLUNTEER', 'SECRETARY', 'PROFESSIONAL', 'COORDINATOR', 'ADMIN'];
      return order.indexOf(roleState.min) >= order.indexOf(role);
    },
  }),
}));

const downloadReceipt = vi.fn();
vi.mock('../../api/donations', () => ({
  downloadReceipt: (...args: unknown[]) =>
    downloadReceipt(...args) as Promise<{ url: string; expires_at: string }>,
}));

import { DonationDetailPage } from '../DonationDetailPage';

const DONATION: DonationDetail = {
  id: 'd-1',
  donor_person_id: null,
  campaign_id: null,
  campus_id: 'c-1',
  donation_type: 'FINANCIAL',
  amount: 150,
  currency: 'BRL',
  item_description: null,
  donation_date: '2026-07-01',
  receipt_number: null,
  receipt_issued_at: null,
  notes: null,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  donor_name: 'Maria Silva',
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/donations/d-1']}>
      <Routes>
        <Route path="/donations/:id" element={<DonationDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('DonationDetailPage receipt', () => {
  beforeEach(() => {
    donationState.donation = DONATION;
    donationState.isLoading = false;
    donationState.error = null;
    roleState.min = 'COORDINATOR';
    downloadReceipt.mockReset();
    vi.spyOn(window, 'open').mockImplementation(() => null);
  });

  it('shows the receipt button for a coordinator', () => {
    renderPage();
    expect(
      screen.getByRole('button', { name: /baixar recibo/i }),
    ).toBeInTheDocument();
  });

  it('hides the receipt button below coordinator', () => {
    roleState.min = 'SECRETARY';
    renderPage();
    expect(
      screen.queryByRole('button', { name: /baixar recibo/i }),
    ).not.toBeInTheDocument();
  });

  it('downloads the receipt and navigates to the presigned url', async () => {
    const user = userEvent.setup();
    downloadReceipt.mockResolvedValue({
      url: 'https://storage.example.com/receipts/d-1.pdf',
      expires_at: '2026-07-06T12:45:00Z',
    });
    renderPage();
    await user.click(screen.getByRole('button', { name: /baixar recibo/i }));
    await waitFor(() => expect(downloadReceipt).toHaveBeenCalledWith('d-1'));
    await waitFor(() =>
      expect(window.open).toHaveBeenCalledWith(
        'https://storage.example.com/receipts/d-1.pdf',
        '_blank',
        'noopener,noreferrer',
      ),
    );
  });

  it('surfaces the receipt number when the receipt has been issued', () => {
    donationState.donation = {
      ...DONATION,
      receipt_number: 'REC-2026-1A2B3C4D',
      receipt_issued_at: '2026-07-06T12:40:00Z',
    };
    renderPage();
    expect(screen.getByText('REC-2026-1A2B3C4D')).toBeInTheDocument();
  });
});
