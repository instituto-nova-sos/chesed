import { apiClient } from './client';
import type {
  Donation,
  DonationDetail,
  DonationListResponse,
  DonationInput,
  UpdateDonationInput,
} from '../types';

export interface ListDonationsParams {
  page?: number;
  per_page?: number;
  donation_type?: string;
  campaign_id?: string;
}

export function listDonations(params: ListDonationsParams = {}): Promise<DonationListResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  if (params.donation_type) searchParams.set('donation_type', params.donation_type);
  if (params.campaign_id) searchParams.set('campaign_id', params.campaign_id);
  const qs = searchParams.toString();
  return apiClient<DonationListResponse>(`/donations${qs ? `?${qs}` : ''}`);
}

export function getDonation(id: string): Promise<DonationDetail> {
  return apiClient<DonationDetail>(`/donations/${id}`);
}

export function createDonation(input: DonationInput): Promise<Donation> {
  return apiClient<Donation>('/donations', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateDonation(id: string, input: UpdateDonationInput): Promise<Donation> {
  return apiClient<Donation>(`/donations/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}
