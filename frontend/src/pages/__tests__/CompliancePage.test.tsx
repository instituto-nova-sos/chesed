import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { ComplianceReport } from '../../types/compliance';
import '../../components/charts/test-helpers';

const reportState: {
  report: ComplianceReport | null;
  isLoading: boolean;
  error: string | null;
  generate: ReturnType<typeof vi.fn>;
} = {
  report: null,
  isLoading: false,
  error: null,
  generate: vi.fn(),
};

vi.mock('../../hooks/useComplianceReport', () => ({
  useComplianceReport: () => ({ ...reportState }),
}));

const offlineState = { isOffline: false };
vi.mock('../../hooks/useOfflineStatus', () => ({
  useOfflineStatus: () => offlineState,
}));

const downloadComplianceCSV = vi.fn();
vi.mock('../../api/compliance', () => ({
  downloadComplianceCSV: (...args: unknown[]) =>
    downloadComplianceCSV(...args) as Promise<Blob>,
}));

import { CompliancePage } from '../CompliancePage';

const REPORT: ComplianceReport = {
  period: { start: '2020-01-01', end: '2030-12-31' },
  consents_by_type: { DATA_PROCESSING: 12, IMAGE_USAGE: 4 },
  active_consents: 14,
  revoked_consents: 2,
  anonymized_subjects: 3,
  data_subjects: 40,
  documents_stored: 9,
};

function renderPage() {
  return render(
    <MemoryRouter>
      <CompliancePage />
    </MemoryRouter>,
  );
}

describe('CompliancePage', () => {
  beforeEach(() => {
    reportState.report = null;
    reportState.isLoading = false;
    reportState.error = null;
    reportState.generate = vi.fn();
    offlineState.isOffline = false;
    downloadComplianceCSV.mockReset();
  });

  it('renders the date-range filter and export button', () => {
    renderPage();
    expect(
      screen.getByRole('button', { name: /gerar relatório/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /exportar csv/i }),
    ).toBeInTheDocument();
  });

  it('renders KPI metrics from the report', () => {
    reportState.report = REPORT;
    renderPage();
    expect(screen.getByText('14')).toBeInTheDocument(); // active consents
    expect(screen.getByText('3')).toBeInTheDocument(); // anonymized subjects
    expect(screen.getByText('40')).toBeInTheDocument(); // data subjects
  });

  it('submits the filter, triggering a report query', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(screen.getByRole('button', { name: /gerar relatório/i }));
    expect(reportState.generate).toHaveBeenCalled();
  });

  it('downloads the CSV when the export button is clicked', async () => {
    const user = userEvent.setup();
    reportState.report = REPORT;
    downloadComplianceCSV.mockResolvedValue(new Blob(['metric,value']));
    renderPage();
    await user.click(screen.getByRole('button', { name: /exportar csv/i }));
    expect(downloadComplianceCSV).toHaveBeenCalled();
  });

  it('shows an offline message instead of querying when offline', () => {
    offlineState.isOffline = true;
    renderPage();
    expect(screen.getByText(/offline/i)).toBeInTheDocument();
  });
});
