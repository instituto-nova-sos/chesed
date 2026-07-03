import { apiClient } from './client';
import type {
  Campaign,
  CampaignDetail,
  CampaignListResponse,
  CampaignInput,
  UpdateCampaignInput,
  AddTeamMemberInput,
  CampaignTeamMember,
  CampaignMetrics,
} from '../types';

export interface ListCampaignsParams {
  page?: number;
  per_page?: number;
  status?: string;
  campaign_type?: string;
}

export function listCampaigns(params: ListCampaignsParams = {}): Promise<CampaignListResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  if (params.status) searchParams.set('status', params.status);
  if (params.campaign_type) searchParams.set('campaign_type', params.campaign_type);
  const qs = searchParams.toString();
  return apiClient<CampaignListResponse>(`/campaigns${qs ? `?${qs}` : ''}`);
}

export function getCampaign(id: string): Promise<CampaignDetail> {
  return apiClient<CampaignDetail>(`/campaigns/${id}`);
}

export function createCampaign(input: CampaignInput): Promise<Campaign> {
  return apiClient<Campaign>('/campaigns', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateCampaign(id: string, input: UpdateCampaignInput): Promise<Campaign> {
  return apiClient<Campaign>(`/campaigns/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export function addTeamMember(
  campaignId: string,
  input: AddTeamMemberInput,
): Promise<CampaignTeamMember> {
  return apiClient<CampaignTeamMember>(`/campaigns/${campaignId}/team`, {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function removeTeamMember(campaignId: string, personId: string): Promise<void> {
  return apiClient<void>(`/campaigns/${campaignId}/team/${personId}`, {
    method: 'DELETE',
  });
}

export function getCampaignMetrics(id: string): Promise<CampaignMetrics> {
  return apiClient<CampaignMetrics>(`/reports/campaigns/${id}`);
}
