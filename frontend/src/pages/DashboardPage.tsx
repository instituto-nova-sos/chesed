import { Link } from 'react-router-dom';
import { Alert } from '../components/ui/Alert';
import { MonthlyTrendChart } from '../components/charts/MonthlyTrendChart';
import { StatusDonutChart } from '../components/charts/StatusDonutChart';
import { useAuth } from '../hooks/useAuth';
import { useDashboard } from '../hooks/useDashboard';
import type { DashboardMetrics } from '../types';

interface KpiCardProps {
  label: string;
  value: number | string;
  href: string;
  accent: string;
}

function KpiCard({ label, value, href, accent }: KpiCardProps) {
  return (
    <Link
      to={href}
      className="block rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition hover:border-gray-300 hover:shadow"
    >
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
        {label}
      </p>
      <p className={`mt-2 text-2xl font-semibold ${accent}`}>{value}</p>
    </Link>
  );
}

function KpiGrid({
  metrics,
  isLoading,
}: {
  metrics: DashboardMetrics | null;
  isLoading: boolean;
}) {
  const show = (value: number | undefined) =>
    isLoading ? '...' : (value ?? '—');
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
      <KpiCard
        label="Pessoas cadastradas"
        value={show(metrics?.total_persons)}
        href="/persons"
        accent="text-blue-600"
      />
      <KpiCard
        label="Atendimentos no mês"
        value={show(metrics?.attendances_this_month)}
        href="/attendances"
        accent="text-blue-700"
      />
      <KpiCard
        label="Agendados"
        value={show(metrics?.upcoming_scheduled)}
        href="/attendances"
        accent="text-amber-600"
      />
      <KpiCard
        label="Campanhas ativas"
        value={show(metrics?.active_campaigns)}
        href="/campaigns"
        accent="text-purple-600"
      />
    </div>
  );
}

function ChartCard({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h2 className="mb-3 text-sm font-semibold text-gray-800">{title}</h2>
      {children}
    </section>
  );
}

function Shortcuts() {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h2 className="text-sm font-medium text-gray-900">Atalhos</h2>
      <div className="mt-3 flex flex-wrap gap-2">
        <Link
          to="/persons"
          className="rounded-lg bg-blue-50 px-3 py-2 text-sm font-medium text-blue-700 hover:bg-blue-100"
        >
          Pessoas
        </Link>
        <Link
          to="/triages"
          className="rounded-lg bg-purple-50 px-3 py-2 text-sm font-medium text-purple-700 hover:bg-purple-100"
        >
          Triagens
        </Link>
        <Link
          to="/attendances"
          className="rounded-lg bg-amber-50 px-3 py-2 text-sm font-medium text-amber-700 hover:bg-amber-100"
        >
          Atendimentos
        </Link>
      </div>
    </div>
  );
}

export function DashboardPage() {
  const { email, roles } = useAuth();
  const { data, isLoading, error } = useDashboard();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-800">Dashboard</h1>
        <p className="mt-1 text-sm text-gray-500">
          {email ? `Bem-vindo, ${email}` : 'Bem-vindo ao Chesed'}
          {roles.length > 0 && (
            <span className="ml-2 text-gray-400">· {roles.join(', ')}</span>
          )}
        </p>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      <KpiGrid metrics={data} isLoading={isLoading} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ChartCard title="Atendimentos por mês">
          <MonthlyTrendChart data={data?.recent_months ?? []} />
        </ChartCard>
        <ChartCard title="Atendimentos por status">
          <StatusDonutChart data={data?.attendances_by_status ?? {}} />
        </ChartCard>
      </div>

      <Shortcuts />
    </div>
  );
}
