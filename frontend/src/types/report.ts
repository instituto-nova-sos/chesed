export interface ReportPeriod {
  start: string;
  end: string;
}

export interface ServiceTypeCount {
  service_type: string;
  count: number;
}

export interface MonthCount {
  month: string;
  count: number;
}

export interface ProfessionalCount {
  professional_id: string;
  professional_name: string;
  count: number;
}

export interface AttendanceReport {
  period: ReportPeriod;
  total_attendances: number;
  unique_persons: number;
  by_status: Record<string, number>;
  by_service_type: ServiceTypeCount[];
  by_month: MonthCount[];
  by_professional: ProfessionalCount[];
}

export interface AttendanceReportParams {
  start: string;
  end: string;
  service_type_id?: string;
  campaign_id?: string;
  professional_id?: string;
}

export interface DashboardMetrics {
  total_persons: number;
  attendances_this_month: number;
  upcoming_scheduled: number;
  active_campaigns: number;
  attendances_by_status: Record<string, number>;
  recent_months: MonthCount[];
}
