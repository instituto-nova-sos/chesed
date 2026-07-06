import { useAuthStore } from '../store/authStore';
import { ApiError, apiClient } from './client';
import type {
  AttendanceReport,
  AttendanceReportParams,
  DashboardMetrics,
} from '../types';

const API_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1';

export function getDashboard(): Promise<DashboardMetrics> {
  return apiClient<DashboardMetrics>('/reports/dashboard');
}

export function getAttendanceReport(
  params: AttendanceReportParams,
): Promise<AttendanceReport> {
  const search = new URLSearchParams({ start: params.start, end: params.end });
  if (params.service_type_id) search.set('service_type_id', params.service_type_id);
  if (params.campaign_id) search.set('campaign_id', params.campaign_id);
  if (params.professional_id) search.set('professional_id', params.professional_id);
  return apiClient<AttendanceReport>(`/reports/attendances?${search.toString()}`);
}

/**
 * Downloads the attendance CSV export and returns a Blob the caller can
 * persist via a temporary anchor element. Bearer token is attached manually
 * because the response is binary, not JSON.
 */
export async function downloadAttendancesCSV(
  params: AttendanceReportParams,
): Promise<Blob> {
  const token = await useAuthStore.getState().getToken();
  const search = new URLSearchParams({
    start: params.start,
    end: params.end,
    format: 'csv',
  });

  const response = await fetch(
    `${API_URL}/reports/attendances/export?${search.toString()}`,
    {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    },
  );

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(response.status, body);
  }
  return response.blob();
}

export function suggestedCSVFilename(params: AttendanceReportParams): string {
  return `attendances_${params.start}_${params.end}.csv`;
}
