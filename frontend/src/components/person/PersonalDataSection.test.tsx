import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useForm } from 'react-hook-form';
import { PersonalDataSection } from './PersonalDataSection';

function Harness() {
  const form = useForm({
    defaultValues: {
      full_name: '',
      nationality: 'USA',
      document_type: 'SSN',
      document_number: '',
    },
  });
  return <PersonalDataSection form={form} />;
}

describe('PersonalDataSection document placeholder', () => {
  it('shows a type-appropriate placeholder for the selected document type', async () => {
    render(<Harness />);

    expect(screen.getByPlaceholderText('000-00-0000')).toBeInTheDocument();

    const typeSelect = screen
      .getAllByRole('combobox')
      .find((el) =>
        Array.from(el.querySelectorAll('option')).some((o) => o.textContent === 'Passaporte'),
      );
    if (!typeSelect) throw new Error('document type select not found');
    await userEvent.selectOptions(typeSelect, 'PASSPORT');

    expect(screen.getByPlaceholderText('AB123456')).toBeInTheDocument();
  });
});
