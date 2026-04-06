import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { LoadingScreen } from './components/ui/LoadingScreen';

describe('LoadingScreen', () => {
  it('renders loading text', () => {
    render(<LoadingScreen />);
    expect(screen.getByText('Carregando...')).toBeInTheDocument();
  });
});
