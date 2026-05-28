import { apiClient } from './client';
import type {
  Attendance,
  AttendanceDetail,
  AttendanceListResponse,
  AttendanceStatus,
  CreateAttendanceInput,
  TransitionAttendanceInput,
  UpdateAttendanceNotesInput,
} from '../types';

export interface ListAttendancesParams {
  page?: number;
  per_page?: number;
  person_id?: string;
  status?: AttendanceStatus;
  from?: string;
  to?: string;
}

export function listAttendances(
  params: ListAttendancesParams = {},
): Promise<AttendanceListResponse> {
  const searchParams = new URLSearchParams();
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  if (params.person_id) searchParams.set('person_id', params.person_id);
  if (params.status) searchParams.set('status', params.status);
  if (params.from) searchParams.set('from', params.from);
  if (params.to) searchParams.set('to', params.to);
  const qs = searchParams.toString();
  return apiClient<AttendanceListResponse>(`/attendances${qs ? `?${qs}` : ''}`);
}

export function getAttendance(id: string): Promise<AttendanceDetail> {
  return apiClient<AttendanceDetail>(`/attendances/${id}`);
}

export function createAttendance(input: CreateAttendanceInput): Promise<Attendance> {
  return apiClient<Attendance>('/attendances', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function transitionAttendance(
  id: string,
  input: TransitionAttendanceInput,
): Promise<Attendance> {
  return apiClient<Attendance>(`/attendances/${id}/transitions`, {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateAttendanceNotes(
  id: string,
  input: UpdateAttendanceNotesInput,
): Promise<Attendance> {
  return apiClient<Attendance>(`/attendances/${id}/notes`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  });
}
