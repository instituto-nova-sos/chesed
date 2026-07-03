import type { Pagination } from './person';

export type AttendanceStatus =
  | 'SCHEDULED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'CANCELLED';

export interface Attendance {
  id: string;
  person_id: string;
  triage_id?: string;
  campaign_id?: string;
  campus_id: string;
  service_type_id: string;
  professional_id: string;
  status: AttendanceStatus;
  attendance_date: string;
  observations?: string;
  recommendations?: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface AttendanceTransition {
  id: string;
  attendance_id: string;
  from_status: AttendanceStatus;
  to_status: AttendanceStatus;
  reason?: string;
  transitioned_by: string;
  transitioned_at: string;
}

export interface AttendanceDetail extends Attendance {
  transitions: AttendanceTransition[];
}

export interface AttendanceListItem {
  id: string;
  person_id: string;
  person_name: string;
  service_type: string;
  status: AttendanceStatus;
  attendance_date: string;
}

export interface AttendanceListResponse {
  data: AttendanceListItem[];
  pagination: Pagination;
}

export interface CreateAttendanceInput {
  person_id: string;
  triage_id?: string;
  campaign_id?: string;
  service_type_id: string;
  professional_id: string;
  attendance_date?: string;
  observations?: string;
  recommendations?: string;
}

export interface TransitionAttendanceInput {
  to: AttendanceStatus;
  reason?: string;
}

export interface UpdateAttendanceNotesInput {
  observations?: string;
  recommendations?: string;
}
