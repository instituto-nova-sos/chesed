import type { Pagination } from './person';

export interface Campaign {
  id: string;
  name: string;
  description?: string;
  campaign_type: string;
  status: string;
  start_date: string;
  end_date?: string;
  location_name?: string;
  location_address?: string;
  campus_id: string;
  coordinator_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CampaignTeamMember {
  person_id: string;
  person_name: string;
  role_in_campaign: string;
  assigned_at: string;
}

export interface CampaignDetail extends Campaign {
  team: CampaignTeamMember[];
}

export interface CampaignListItem {
  id: string;
  name: string;
  campaign_type: string;
  status: string;
  start_date: string;
  end_date?: string;
  location_name?: string;
}

export interface CampaignListResponse {
  data: CampaignListItem[];
  pagination: Pagination;
}

export interface CampaignInput {
  name: string;
  description?: string;
  campaign_type: string;
  start_date: string;
  end_date?: string;
  location_name?: string;
  location_address?: string;
  coordinator_id?: string;
}

export interface UpdateCampaignInput extends CampaignInput {
  status: string;
}

export interface AddTeamMemberInput {
  person_id: string;
  role_in_campaign: string;
}

export interface CampaignMetrics {
  campaign_id: string;
  campaign_name: string;
  status: string;
  period: { start_date: string; end_date?: string };
  triage_count: number;
  attendance_total: number;
  attendances_by_status: Record<string, number>;
  team_size: number;
}
