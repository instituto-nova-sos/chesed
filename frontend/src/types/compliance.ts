export interface ComplianceReportParams {
  start: string;
  end: string;
}

export interface ComplianceReport {
  period: { start: string; end: string };
  consents_by_type: Record<string, number>;
  active_consents: number;
  revoked_consents: number;
  anonymized_subjects: number;
  data_subjects: number;
  documents_stored: number;
}
