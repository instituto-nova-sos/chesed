import { useMemo, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { ChartEmpty } from '../components/charts/ChartEmpty';
import { useComplianceReport } from '../hooks/useComplianceReport';
import { useOfflineStatus } from '../hooks/useOfflineStatus';
import {
  downloadComplianceCSV,
  suggestedComplianceFilename,
} from '../api/compliance';
import type { ComplianceReport } from '../types/compliance';

const BAR_COLOR = '#0d9488'; // teal-600
const GRID_COLOR = '#e5e7eb'; // gray-200
const AXIS_COLOR = '#6b7280'; // gray-500

function defaultRange(): { start: string; end: string } {
  const today = new Date();
  const end = today.toISOString().slice(0, 10);
  const start = new Date(today.getFullYear() - 1, today.getMonth(), today.getDate())
    .toISOString()
    .slice(0, 10);
  return { start, end };
}

function triggerDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function MetricCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="text-xs uppercase text-gray-500">{label}</div>
      <div className="mt-1 text-2xl font-semibold text-gray-900">{value}</div>
    </div>
  );
}

function ConsentTypeChart({ data }: { data: Record<string, number> }) {
  const rows = Object.entries(data).map(([type, count]) => ({ type, count }));
  if (rows.length === 0) return <ChartEmpty />;
  return (
    <div data-testid="consent-type-chart" className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={rows} layout="vertical" margin={{ left: 8, right: 16 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={GRID_COLOR} horizontal={false} />
          <XAxis type="number" stroke={AXIS_COLOR} fontSize={12} allowDecimals={false} />
          <YAxis type="category" dataKey="type" stroke={AXIS_COLOR} fontSize={12} width={120} />
          <Tooltip cursor={{ fill: 'rgba(13, 148, 136, 0.08)' }} />
          <Bar dataKey="count" fill={BAR_COLOR} radius={[0, 4, 4, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

interface ComplianceFiltersProps {
  range: { start: string; end: string };
  onRangeChange: (key: 'start' | 'end', value: string) => void;
  onSubmit: (event: React.FormEvent) => void;
  onDownload: () => void;
  isLoading: boolean;
  isDownloading: boolean;
  canDownload: boolean;
}

function ComplianceFilters(props: ComplianceFiltersProps) {
  return (
    <form
      onSubmit={props.onSubmit}
      className="grid grid-cols-1 gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:grid-cols-2 lg:grid-cols-3"
    >
      <Input
        id="compliance-start"
        label="Início"
        type="date"
        value={props.range.start}
        onChange={(e) => props.onRangeChange('start', e.target.value)}
        required
      />
      <Input
        id="compliance-end"
        label="Fim"
        type="date"
        value={props.range.end}
        onChange={(e) => props.onRangeChange('end', e.target.value)}
        required
      />
      <div className="flex items-end gap-3">
        <Button type="submit" isLoading={props.isLoading}>
          Gerar relatório
        </Button>
        <Button
          type="button"
          variant="secondary"
          onClick={props.onDownload}
          disabled={!props.canDownload || props.isDownloading}
          isLoading={props.isDownloading}
        >
          Exportar CSV
        </Button>
      </div>
    </form>
  );
}

function ReportContent({ report }: { report: ComplianceReport }) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <MetricCard label="Consentimentos ativos" value={report.active_consents} />
        <MetricCard label="Consentimentos revogados" value={report.revoked_consents} />
        <MetricCard label="Titulares anonimizados" value={report.anonymized_subjects} />
        <MetricCard label="Titulares de dados" value={report.data_subjects} />
        <MetricCard label="Documentos armazenados" value={report.documents_stored} />
      </div>
      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="mb-3 text-sm font-semibold text-gray-800">
          Consentimentos por tipo
        </h2>
        <ConsentTypeChart data={report.consents_by_type} />
      </section>
    </div>
  );
}

export function CompliancePage() {
  const initial = useMemo(() => defaultRange(), []);
  const [range, setRange] = useState(initial);
  const { isOffline } = useOfflineStatus();
  const { report, isLoading, error, generate } = useComplianceReport(
    isOffline ? undefined : initial,
  );
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  function handleGenerate(event: React.FormEvent) {
    event.preventDefault();
    if (!range.start || !range.end) return;
    setDownloadError(null);
    generate({ start: range.start, end: range.end });
  }

  async function handleDownload() {
    if (!report) return;
    setDownloading(true);
    setDownloadError(null);
    try {
      const params = { start: range.start, end: range.end };
      const blob = await downloadComplianceCSV(params);
      triggerDownload(blob, suggestedComplianceFilename(params));
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : 'Falha ao exportar CSV');
    } finally {
      setDownloading(false);
    }
  }

  if (isOffline) {
    return (
      <div className="space-y-6">
        <h1 className="text-lg font-semibold text-gray-900">Conformidade (LGPD)</h1>
        <Alert variant="warning">
          O relatório de conformidade está indisponível offline. Reconecte-se para
          consultar as métricas atualizadas.
        </Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold text-gray-900">Conformidade (LGPD)</h1>
      </header>

      <ComplianceFilters
        range={range}
        onRangeChange={(key, value) => setRange((r) => ({ ...r, [key]: value }))}
        onSubmit={handleGenerate}
        onDownload={handleDownload}
        isLoading={isLoading}
        isDownloading={downloading}
        canDownload={report !== null}
      />

      {error && <Alert variant="error">{error}</Alert>}
      {downloadError && <Alert variant="error">{downloadError}</Alert>}

      {isLoading ? (
        <LoadingScreen />
      ) : report ? (
        <ReportContent report={report} />
      ) : (
        <EmptyState
          title="Selecione um período"
          description="Escolha um intervalo de datas para gerar o relatório de conformidade."
        />
      )}
    </div>
  );
}
