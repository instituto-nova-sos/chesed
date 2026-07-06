import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import './test-helpers';
import { MonthlyTrendChart } from './MonthlyTrendChart';
import type { MonthCount } from '../../types';

const DATA: MonthCount[] = [
  { month: '2026-06', count: 9 },
  { month: '2026-07', count: 8 },
];

describe('MonthlyTrendChart', () => {
  it('renders a chart container when data is present', () => {
    render(<MonthlyTrendChart data={DATA} />);
    expect(screen.getByTestId('monthly-trend-chart')).toBeInTheDocument();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('shows the empty fallback when data is empty', () => {
    render(<MonthlyTrendChart data={[]} />);
    expect(screen.getByRole('status')).toHaveTextContent(/sem dados no período/i);
    expect(screen.queryByTestId('monthly-trend-chart')).not.toBeInTheDocument();
  });
});
