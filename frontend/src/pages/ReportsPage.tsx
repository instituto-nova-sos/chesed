import { useEffect, useMemo, useState } from 'react';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Alert } from '../components/ui/Alert';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { MonthlyTrendChart } from '../components/charts/MonthlyTrendChart';
import { StatusDonutChart } from '../components/charts/StatusDonutChart';
import { ServiceTypeBarChart } from '../components/charts/ServiceTypeBarChart';
import { useAttendanceReport } from '../hooks/useAttendanceReport';
import { useLinkableCampaigns } from '../hooks/useCampaigns';
import {
  downloadAttendancesCSV,
  suggestedCSVFilename,
} from '../api/reports';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
import type { AttendanceReport, ProfessionalCount } from '../types';

interface Option {
  value: string;
  label: string;
}

function defaultRange(): { start: string; end: string } {
  const today = new Date();
  const end = today.toISOString().slice(0, 10);
  const first = new Date(today.getFullYear(), today.getMonth(), 1)
    .toISOString()
    .slice(0, 10);
  return { start: first, end };
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

function useServiceTypeOptions(): Option[] {
  const [types, setTypes] = useState<ServiceType[]>([]);
  useEffect(() => {
    let cancelled = false;
    listServiceTypes()
      .then((res) => {
        if (!cancelled) setTypes(res.data);
      })
      .catch(() => {
        // Optional filter: an empty list degrades to "all service types".
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return types.map((t) => ({ value: t.id, label: t.name }));
}

interface ReportFiltersProps {
  range: { start: string; end: string };
  filters: { service_type_id: string; campaign_id: string; professional_id: string };
  serviceTypeOptions: Option[];
  campaignOptions: Option[];
  professionalOptions: Option[];
  onRangeChange: (key: 'start' | 'end', value: string) => void;
  onFilterChange: (key: keyof ReportFiltersProps['filters'], value: string) => void;
  onSubmit: (event: React.FormEvent) => void;
  onDownload: () => void;
  isLoading: boolean;
  isDownloading: boolean;
  canDownload: boolean;
}

function ReportFilters(props: ReportFiltersProps) {
  return (
    <form
      onSubmit={props.onSubmit}
      className="grid grid-cols-1 gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:grid-cols-2 lg:grid-cols-3"
    >
      <Input
        label="Início"
        type="date"
        value={props.range.start}
        onChange={(e) => props.onRangeChange('start', e.target.value)}
        required
      />
      <Input
        label="Fim"
        type="date"
        value={props.range.end}
        onChange={(e) => props.onRangeChange('end', e.target.value)}
        required
      />
      <Select
        label="Tipo de serviço"
        placeholder="Todos"
        options={props.serviceTypeOptions}
        value={props.filters.service_type_id}
        onChange={(e) => props.onFilterChange('service_type_id', e.target.value)}
      />
      <Select
        label="Campanha"
        placeholder="Todas"
        options={props.campaignOptions}
        value={props.filters.campaign_id}
        onChange={(e) => props.onFilterChange('campaign_id', e.target.value)}
      />
      <Select
        label="Profissional"
        placeholder="Todos"
        options={props.professionalOptions}
        value={props.filters.professional_id}
        onChange={(e) => props.onFilterChange('professional_id', e.target.value)}
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

function MetricCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="text-xs uppercase text-gray-500">{label}</div>
      <div className="mt-1 text-2xl font-semibold text-gray-900">{value}</div>
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h2 className="mb-3 text-sm font-semibold text-gray-800">{title}</h2>
      {children}
    </section>
  );
}

function ProfessionalTable({ rows }: { rows: ProfessionalCount[] }) {
  if (rows.length === 0) {
    return <p className="text-sm text-gray-500">Nenhum atendimento no período.</p>;
  }
  return (
    <ul className="divide-y divide-gray-100 text-sm">
      {rows.map((row) => (
        <li
          key={row.professional_id}
          className="flex items-center justify-between py-2"
        >
          <span className="text-gray-700">{row.professional_name}</span>
          <span className="font-medium text-gray-900">{row.count}</span>
        </li>
      ))}
    </ul>
  );
}

function ReportResult({
  report,
  isLoading,
}: {
  report: AttendanceReport | null;
  isLoading: boolean;
}) {
  if (isLoading) return <LoadingScreen />;
  if (report) return <ReportContent report={report} />;
  return (
    <EmptyState
      title="Selecione um período"
      description="Escolha um intervalo de datas para gerar o relatório de atendimentos."
    />
  );
}

function ReportContent({ report }: { report: AttendanceReport }) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Atendimentos totais" value={report.total_attendances} />
        <MetricCard label="Pessoas únicas" value={report.unique_persons} />
        <MetricCard label="Concluídos" value={report.by_status.COMPLETED ?? 0} />
        <MetricCard label="Em andamento" value={report.by_status.IN_PROGRESS ?? 0} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card title="Atendimentos por mês">
          <MonthlyTrendChart data={report.by_month} />
        </Card>
        <Card title="Atendimentos por status">
          <StatusDonutChart data={report.by_status} />
        </Card>
        <Card title="Atendimentos por tipo de serviço">
          <ServiceTypeBarChart data={report.by_service_type} />
        </Card>
        <Card title="Atendimentos por profissional">
          <ProfessionalTable rows={report.by_professional} />
        </Card>
      </div>
    </div>
  );
}

export function ReportsPage() {
  const initial = useMemo(() => defaultRange(), []);
  const [range, setRange] = useState(initial);
  const [filters, setFilters] = useState({
    service_type_id: '',
    campaign_id: '',
    professional_id: '',
  });
  const { report, isLoading, error, generate } = useAttendanceReport(initial);
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  const serviceTypeOptions = useServiceTypeOptions();
  const campaignOptions = useLinkableCampaigns().map((c) => ({
    value: c.id,
    label: c.name,
  }));
  const professionalOptions: Option[] = (report?.by_professional ?? []).map(
    (p) => ({ value: p.professional_id, label: p.professional_name }),
  );

  function handleGenerate(event: React.FormEvent) {
    event.preventDefault();
    if (!range.start || !range.end) return;
    generate({
      start: range.start,
      end: range.end,
      service_type_id: filters.service_type_id || undefined,
      campaign_id: filters.campaign_id || undefined,
      professional_id: filters.professional_id || undefined,
    });
  }

  async function handleDownload() {
    if (!report) return;
    setDownloading(true);
    setDownloadError(null);
    try {
      const params = { start: range.start, end: range.end };
      const blob = await downloadAttendancesCSV(params);
      triggerDownload(blob, suggestedCSVFilename(params));
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : 'Falha ao exportar CSV');
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-lg font-semibold text-gray-900">Relatórios</h1>
      </header>

      <ReportFilters
        range={range}
        filters={filters}
        serviceTypeOptions={serviceTypeOptions}
        campaignOptions={campaignOptions}
        professionalOptions={professionalOptions}
        onRangeChange={(key, value) => setRange((r) => ({ ...r, [key]: value }))}
        onFilterChange={(key, value) =>
          setFilters((f) => ({ ...f, [key]: value }))
        }
        onSubmit={handleGenerate}
        onDownload={handleDownload}
        isLoading={isLoading}
        isDownloading={downloading}
        canDownload={report !== null}
      />

      {error && <Alert variant="error">{error}</Alert>}
      {downloadError && <Alert variant="error">{downloadError}</Alert>}

      <ReportResult report={report} isLoading={isLoading} />
    </div>
  );
}
