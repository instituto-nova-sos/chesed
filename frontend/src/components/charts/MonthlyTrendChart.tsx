import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { MonthCount } from '../../types';
import { ChartEmpty } from './ChartEmpty';

const BAR_COLOR = '#2563eb'; // blue-600
const GRID_COLOR = '#e5e7eb'; // gray-200
const AXIS_COLOR = '#6b7280'; // gray-500

export function MonthlyTrendChart({ data }: { data: MonthCount[] }) {
  if (data.length === 0) return <ChartEmpty />;

  return (
    <div data-testid="monthly-trend-chart" className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={GRID_COLOR} vertical={false} />
          <XAxis dataKey="month" stroke={AXIS_COLOR} fontSize={12} tickMargin={8} />
          <YAxis stroke={AXIS_COLOR} fontSize={12} allowDecimals={false} width={40} />
          <Tooltip cursor={{ fill: 'rgba(37, 99, 235, 0.08)' }} />
          <Bar dataKey="count" fill={BAR_COLOR} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
