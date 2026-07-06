import { useAuthStore } from '../store/authStore';
import { ApiError, apiClient } from './client';
import type { ComplianceReport, ComplianceReportParams } from '../types/compliance';

const API_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1';

export function getComplianceReport(
  params: ComplianceReportParams,
): Promise<ComplianceReport> {
  const search = new URLSearchParams({ start: params.start, end: params.end });
  return apiClient<ComplianceReport>(`/reports/compliance?${search.toString()}`);
}

/**
 * Downloads the compliance CSV export and returns a Blob the caller can persist
 * via a temporary anchor. Bearer token is attached manually because the
 * response is binary, not JSON (mirrors reports.ts downloadAttendancesCSV).
 */
export async function downloadComplianceCSV(
  params: ComplianceReportParams,
): Promise<Blob> {
  const token = await useAuthStore.getState().getToken();
  const search = new URLSearchParams({
    start: params.start,
    end: params.end,
    format: 'csv',
  });

  const response = await fetch(
    `${API_URL}/reports/compliance/export?${search.toString()}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} },
  );

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(response.status, body);
  }
  return response.blob();
}

export function suggestedComplianceFilename(params: ComplianceReportParams): string {
  return `compliance_${params.start}_${params.end}.csv`;
}
