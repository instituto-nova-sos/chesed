import { useEffect, useState } from 'react';
import type { UseFormRegisterReturn } from 'react-hook-form';
import { Select } from '../ui/Select';
import { Input } from '../ui/Input';
import { REFERRAL_SOURCE_OPTIONS } from '../../utils/brazilStates';

interface ReferralSourceSelectProps {
  registration: UseFormRegisterReturn;
  error?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
}

export function ReferralSourceSelect({
  registration,
  error,
  defaultValue,
  onValueChange,
}: ReferralSourceSelectProps) {
  const isPredefined = REFERRAL_SOURCE_OPTIONS.some(
    (o) => o.value === defaultValue && o.value !== 'Outros',
  );
  const [isOther, setIsOther] = useState(
    defaultValue ? !isPredefined : false,
  );

  // Sync state when defaultValue changes (e.g., form reset on edit)
  useEffect(() => {
    if (defaultValue) {
      const predefined = REFERRAL_SOURCE_OPTIONS.some(
        (o) => o.value === defaultValue && o.value !== 'Outros',
      );
      setIsOther(!predefined);
    } else {
      setIsOther(false);
    }
  }, [defaultValue]);

  const options = REFERRAL_SOURCE_OPTIONS.map((o) => ({
    value: o.value,
    label: o.label,
  }));

  return (
    <div>
      {isOther ? (
        <div className="space-y-2">
          <Input
            label="Fonte de Encaminhamento"
            registration={registration}
            error={error}
            placeholder="Descreva a fonte..."
          />
          <button
            type="button"
            onClick={() => {
              setIsOther(false);
              onValueChange?.('');
            }}
            className="text-xs text-blue-600 hover:underline"
          >
            Escolher da lista
          </button>
        </div>
      ) : (
        <div className="space-y-2">
          <Select
            label="Fonte de Encaminhamento"
            options={options}
            registration={registration}
            error={error}
            placeholder="Selecione..."
            onChange={(e) => {
              if (e.target.value === 'Outros') {
                setIsOther(true);
                onValueChange?.('');
              } else {
                registration.onChange(e);
              }
            }}
          />
        </div>
      )}
    </div>
  );
}
