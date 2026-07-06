import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import './test-helpers';
import { ServiceTypeBarChart } from './ServiceTypeBarChart';
import type { ServiceTypeCount } from '../../types';

const DATA: ServiceTypeCount[] = [
  { service_type: 'LEGAL', count: 12 },
  { service_type: 'MEDICAL', count: 28 },
];

describe('ServiceTypeBarChart', () => {
  it('renders a chart container when data is present', () => {
    render(<ServiceTypeBarChart data={DATA} />);
    expect(screen.getByTestId('service-type-bar-chart')).toBeInTheDocument();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('shows the empty fallback when data is empty', () => {
    render(<ServiceTypeBarChart data={[]} />);
    expect(screen.getByRole('status')).toHaveTextContent(/sem dados no período/i);
    expect(screen.queryByTestId('service-type-bar-chart')).not.toBeInTheDocument();
  });
});
