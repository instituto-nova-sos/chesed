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

export interface AttendanceReport {
  period: ReportPeriod;
  total_attendances: number;
  unique_persons: number;
  by_status: Record<string, number>;
  by_service_type: ServiceTypeCount[];
  by_month: MonthCount[];
}

export interface AttendanceReportParams {
  start: string;
  end: string;
}
