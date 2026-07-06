import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import './test-helpers';
import { StatusDonutChart } from './StatusDonutChart';

describe('StatusDonutChart', () => {
  it('renders a chart container when the status map has entries', () => {
    render(
      <StatusDonutChart data={{ COMPLETED: 30, SCHEDULED: 5, CANCELLED: 1 }} />,
    );
    expect(screen.getByTestId('status-donut-chart')).toBeInTheDocument();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('shows the empty fallback when the status map is empty', () => {
    render(<StatusDonutChart data={{}} />);
    expect(screen.getByRole('status')).toHaveTextContent(/sem dados no período/i);
    expect(screen.queryByTestId('status-donut-chart')).not.toBeInTheDocument();
  });

  it('treats an all-zero status map as empty', () => {
    render(<StatusDonutChart data={{ COMPLETED: 0, SCHEDULED: 0 }} />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });
});
