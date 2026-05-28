import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { listAttendances } from '../api/attendances';
import { listTriages } from '../api/triages';
import { useAuth } from '../hooks/useAuth';

interface Metrics {
  attendancesToday: number;
  attendancesWeek: number;
  attendancesInProgress: number;
  triagesWeek: number;
}

function startOfDay(): string {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d.toISOString();
}

function startOfWeek(): string {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  const day = d.getDay();
  const diff = day === 0 ? 6 : day - 1; // Monday-anchored week
  d.setDate(d.getDate() - diff);
  return d.toISOString();
}

function MetricCard({
  label,
  value,
  href,
  accent,
}: {
  label: string;
  value: number | string;
  href: string;
  accent: string;
}) {
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

export function DashboardPage() {
  const { email, roles } = useAuth();
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const today = startOfDay();
    const weekStart = startOfWeek();
    let cancelled = false;
    setLoading(true);

    Promise.all([
      listAttendances({ from: today, per_page: 1 }),
      listAttendances({ from: weekStart, per_page: 1 }),
      listAttendances({ status: 'IN_PROGRESS', per_page: 1 }),
      listTriages({ from: weekStart, per_page: 1 }),
    ])
      .then(([todayRes, weekRes, inProgressRes, triagesRes]) => {
        if (cancelled) return;
        setMetrics({
          attendancesToday: todayRes.pagination.total,
          attendancesWeek: weekRes.pagination.total,
          attendancesInProgress: inProgressRes.pagination.total,
          triagesWeek: triagesRes.pagination.total,
        });
      })
      .catch(() => {
        if (!cancelled) setMetrics(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

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

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <MetricCard
          label="Atendimentos Hoje"
          value={loading ? '...' : (metrics?.attendancesToday ?? '—')}
          href="/attendances"
          accent="text-blue-600"
        />
        <MetricCard
          label="Atendimentos Semana"
          value={loading ? '...' : (metrics?.attendancesWeek ?? '—')}
          href="/attendances"
          accent="text-blue-700"
        />
        <MetricCard
          label="Em Andamento"
          value={loading ? '...' : (metrics?.attendancesInProgress ?? '—')}
          href="/attendances"
          accent="text-amber-600"
        />
        <MetricCard
          label="Triagens Semana"
          value={loading ? '...' : (metrics?.triagesWeek ?? '—')}
          href="/triages"
          accent="text-purple-600"
        />
      </div>

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
    </div>
  );
}
