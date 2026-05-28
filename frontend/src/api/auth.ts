import { apiClient } from './client';

export interface OnboardingStatus {
  person_id: string | null;
  campus_id: string | null;
  needs_profile_completion: boolean;
  needs_campus_assignment: boolean;
  needs_agreement: boolean;
  roles: string[];
}

export function getOnboardingStatus(): Promise<OnboardingStatus> {
  return apiClient<OnboardingStatus>('/auth/me');
}
