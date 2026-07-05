import type { Pagination } from './person';

export interface Donation {
  id: string;
  donor_person_id: string | null;
  campaign_id: string | null;
  campus_id: string;
  donation_type: string;
  amount: number | null;
  currency: string;
  item_description: string | null;
  donation_date: string;
  receipt_number: string | null;
  receipt_issued_at: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface DonationDetail extends Donation {
  donor_name?: string;
  campaign_name?: string;
}

export interface DonationListItem {
  id: string;
  donation_type: string;
  amount: number | null;
  currency: string;
  donation_date: string;
  donor_name?: string | null;
  campaign_name?: string | null;
}

export interface DonationListResponse {
  data: DonationListItem[];
  pagination: Pagination;
}

export interface DonationInput {
  donation_type: string;
  amount?: number;
  currency?: string;
  item_description?: string;
  donor_person_id?: string;
  campaign_id?: string;
  donation_date?: string;
  notes?: string;
}

export type UpdateDonationInput = DonationInput;
