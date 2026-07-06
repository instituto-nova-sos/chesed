import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { ServiceTypeCount } from '../../types';
import { ChartEmpty } from './ChartEmpty';

const BAR_COLOR = '#0d9488'; // teal-600
const GRID_COLOR = '#e5e7eb'; // gray-200
const AXIS_COLOR = '#6b7280'; // gray-500

export function ServiceTypeBarChart({ data }: { data: ServiceTypeCount[] }) {
  if (data.length === 0) return <ChartEmpty />;

  return (
    <div data-testid="service-type-bar-chart" className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart
          data={data}
          layout="vertical"
          margin={{ top: 8, right: 16, bottom: 0, left: 8 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke={GRID_COLOR} horizontal={false} />
          <XAxis type="number" stroke={AXIS_COLOR} fontSize={12} allowDecimals={false} />
          <YAxis
            type="category"
            dataKey="service_type"
            stroke={AXIS_COLOR}
            fontSize={12}
            width={96}
          />
          <Tooltip cursor={{ fill: 'rgba(13, 148, 136, 0.08)' }} />
          <Bar dataKey="count" fill={BAR_COLOR} radius={[0, 4, 4, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
