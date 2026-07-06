import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts';
import { ChartEmpty } from './ChartEmpty';

const STATUS_LABELS: Record<string, string> = {
  SCHEDULED: 'Agendado',
  IN_PROGRESS: 'Em Andamento',
  COMPLETED: 'Concluído',
  CANCELLED: 'Cancelado',
};

const STATUS_COLORS: Record<string, string> = {
  SCHEDULED: '#3b82f6', // blue-500
  IN_PROGRESS: '#f59e0b', // amber-500
  COMPLETED: '#22c55e', // green-500
  CANCELLED: '#ef4444', // red-500
};

const FALLBACK_COLOR = '#9ca3af'; // gray-400

interface Slice {
  key: string;
  name: string;
  value: number;
  color: string;
}

function toSlices(data: Record<string, number>): Slice[] {
  return Object.entries(data)
    .filter(([, value]) => value > 0)
    .map(([key, value]) => ({
      key,
      name: STATUS_LABELS[key] ?? key,
      value,
      color: STATUS_COLORS[key] ?? FALLBACK_COLOR,
    }));
}

export function StatusDonutChart({ data }: { data: Record<string, number> }) {
  const slices = toSlices(data);
  if (slices.length === 0) return <ChartEmpty />;

  return (
    <div data-testid="status-donut-chart" className="h-56 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={slices}
            dataKey="value"
            nameKey="name"
            innerRadius="55%"
            outerRadius="80%"
            paddingAngle={2}
          >
            {slices.map((slice) => (
              <Cell key={slice.key} fill={slice.color} />
            ))}
          </Pie>
          <Tooltip />
          <Legend
            verticalAlign="bottom"
            height={32}
            wrapperStyle={{ fontSize: 12 }}
          />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
