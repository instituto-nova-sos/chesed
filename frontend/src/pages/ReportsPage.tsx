import { useMemo, useState } from 'react';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { EmptyState } from '../components/ui/EmptyState';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { useAttendanceReport } from '../hooks/useAttendanceReport';
import { downloadAttendancesCSV, suggestedCSVFilename } from '../api/reports';
import type { AttendanceReport } from '../types';

const STATUS_LABELS: Record<string, string> = {
  SCHEDULED: 'Agendado',
  IN_PROGRESS: 'Em Andamento',
  COMPLETED: 'Concluído',
  CANCELLED: 'Cancelado',
};

const STATUS_ORDER = ['SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'];

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

interface ReportFiltersProps {
  start: string;
  end: string;
  onStartChange: (value: string) => void;
  onEndChange: (value: string) => void;
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
      className="flex flex-wrap items-end gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm"
    >
      <Input
        label="Início"
        type="date"
        value={props.start}
        onChange={(e) => props.onStartChange(e.target.value)}
        required
        className="min-w-[10rem]"
      />
      <Input
        label="Fim"
        type="date"
        value={props.end}
        onChange={(e) => props.onEndChange(e.target.value)}
        required
        className="min-w-[10rem]"
      />
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
    </form>
  );
}

export function ReportsPage() {
  const initial = useMemo(() => defaultRange(), []);
  const [start, setStart] = useState(initial.start);
  const [end, setEnd] = useState(initial.end);
  const { report, isLoading, error, generate } = useAttendanceReport(initial);
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  function handleGenerate(event: React.FormEvent) {
    event.preventDefault();
    if (!start || !end) return;
    generate({ start, end });
  }

  async function handleDownload() {
    if (!report) return;
    setDownloading(true);
    setDownloadError(null);
    try {
      const params = { start, end };
      const blob = await downloadAttendancesCSV(params);
      triggerDownload(blob, suggestedCSVFilename(params));
    } catch (err) {
      setDownloadError(
        err instanceof Error ? err.message : 'Falha ao exportar CSV',
      );
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
        start={start}
        end={end}
        onStartChange={setStart}
        onEndChange={setEnd}
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
          description="Escolha um intervalo de datas para gerar o relatório de atendimentos."
        />
      )}
    </div>
  );
}

function ReportContent({ report }: { report: AttendanceReport }) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label="Atendimentos totais"
          value={report.total_attendances}
        />
        <MetricCard label="Pessoas únicas" value={report.unique_persons} />
        <MetricCard
          label="Concluídos"
          value={report.by_status.COMPLETED ?? 0}
        />
        <MetricCard
          label="Em andamento"
          value={report.by_status.IN_PROGRESS ?? 0}
        />
      </div>

      <BreakdownCard title="Por status">
        <ul className="divide-y divide-gray-100 text-sm">
          {STATUS_ORDER.filter((s) => report.by_status[s] !== undefined).map(
            (status) => (
              <li
                key={status}
                className="flex items-center justify-between py-2"
              >
                <span className="text-gray-700">
                  {STATUS_LABELS[status] ?? status}
                </span>
                <span className="font-medium text-gray-900">
                  {report.by_status[status]}
                </span>
              </li>
            ),
          )}
        </ul>
      </BreakdownCard>

      <BreakdownCard title="Por tipo de serviço">
        {report.by_service_type.length === 0 ? (
          <p className="text-sm text-gray-500">
            Nenhum atendimento no período.
          </p>
        ) : (
          <ul className="divide-y divide-gray-100 text-sm">
            {report.by_service_type.map((row) => (
              <li
                key={row.service_type}
                className="flex items-center justify-between py-2"
              >
                <span className="text-gray-700">{row.service_type}</span>
                <span className="font-medium text-gray-900">{row.count}</span>
              </li>
            ))}
          </ul>
        )}
      </BreakdownCard>

      <BreakdownCard title="Por mês">
        {report.by_month.length === 0 ? (
          <p className="text-sm text-gray-500">
            Nenhum atendimento no período.
          </p>
        ) : (
          <ul className="divide-y divide-gray-100 text-sm">
            {report.by_month.map((row) => (
              <li
                key={row.month}
                className="flex items-center justify-between py-2"
              >
                <span className="text-gray-700">{row.month}</span>
                <span className="font-medium text-gray-900">{row.count}</span>
              </li>
            ))}
          </ul>
        )}
      </BreakdownCard>
    </div>
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

function BreakdownCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h2 className="mb-2 text-sm font-semibold text-gray-800">{title}</h2>
      {children}
    </section>
  );
}
