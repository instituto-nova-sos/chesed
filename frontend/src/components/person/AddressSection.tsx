import { useCallback, useEffect, useRef } from 'react';
import type { UseFormReturn } from 'react-hook-form';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { SearchableSelect } from '../ui/SearchableSelect';
import { useCepLookup } from '../../hooks/useCepLookup';
import { useCityLookup } from '../../hooks/useCityLookup';
import { BRAZIL_STATES } from '../../utils/brazilStates';
import { COUNTRIES } from '../../utils/countries';

interface AddressSectionProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: UseFormReturn<any, any, any>;
}

type AddressForm = AddressSectionProps['form'];

const stateOptions = BRAZIL_STATES.map((s) => ({
  value: s.code,
  label: s.name,
}));

const countryOptions = COUNTRIES.map((c) => ({
  value: c.code,
  label: `${c.flag} ${c.name}`,
  searchTerms: `${c.alpha2} ${c.code}`,
}));

interface StateCityFieldsProps {
  form: AddressForm;
  isBrazil: boolean;
  stateValue: string;
  cityOptions: { value: string; label: string }[];
  citiesLoading: boolean;
}

function StateCityFields({
  form,
  isBrazil,
  stateValue,
  cityOptions,
  citiesLoading,
}: StateCityFieldsProps) {
  const { register } = form;

  if (!isBrazil) {
    return (
      <>
        <Input label="Estado / Província" registration={register('address.state')} />
        <Input label="Cidade" registration={register('address.city')} />
      </>
    );
  }

  const cityPlaceholder = citiesLoading
    ? 'Carregando...'
    : stateValue
      ? 'Selecione...'
      : 'Selecione o estado primeiro';

  return (
    <>
      <Select
        label="Estado"
        options={stateOptions}
        registration={register('address.state')}
        placeholder="Selecione..."
      />
      <div className="relative">
        <Select
          label="Cidade"
          options={cityOptions}
          registration={register('address.city')}
          placeholder={cityPlaceholder}
        />
        {citiesLoading && (
          <span className="absolute right-8 top-8 text-xs text-blue-600">
            Carregando...
          </span>
        )}
      </div>
    </>
  );
}

interface AddressAutofillState {
  country: string;
  stateValue: string;
  isBrazil: boolean;
  cepLoading: boolean;
  citiesLoading: boolean;
  cityOptions: { value: string; label: string }[];
}

function useAddressAutofill(form: AddressForm): AddressAutofillState {
  const { setValue, watch } = form;
  const { lookupCep, isLoading: cepLoading } = useCepLookup();
  const { cities, fetchCities, clearCities, isLoading: citiesLoading } =
    useCityLookup();
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const pendingCityRef = useRef<string | null>(null);

  const country = watch('address.country') || 'BRA';
  const stateValue = watch('address.state') || '';
  const zipCode = watch('address.zip_code') || '';
  const isBrazil = country === 'BRA';

  const cityOptions = cities.map((c) => ({ value: c.name, label: c.name }));

  const handleCepLookup = useCallback(
    async (cep: string) => {
      const result = await lookupCep(cep);
      if (result) {
        setValue('address.street', result.street, { shouldValidate: true });
        setValue('address.neighborhood', result.neighborhood, {
          shouldValidate: true,
        });
        setValue('address.state', result.state, { shouldValidate: true });
        pendingCityRef.current = result.city;
        setValue('address.city', result.city, { shouldValidate: true });
      }
    },
    [lookupCep, setValue],
  );

  // CEP auto-lookup (Brazil only)
  useEffect(() => {
    if (!isBrazil) return;
    const digits = zipCode.replace(/\D/g, '');
    if (digits.length === 8) {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => handleCepLookup(digits), 300);
    }
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [zipCode, handleCepLookup, isBrazil]);

  // Fetch cities when state changes (Brazil only)
  useEffect(() => {
    if (isBrazil && stateValue) {
      fetchCities(stateValue);
    } else {
      clearCities();
    }
  }, [isBrazil, stateValue, fetchCities, clearCities]);

  // Re-apply pending city after cities load (from CEP auto-fill)
  useEffect(() => {
    if (pendingCityRef.current && cities.length > 0) {
      const pending = pendingCityRef.current;
      const match = cities.find((c) => c.name === pending);
      if (match) {
        setValue('address.city', match.name, { shouldValidate: true });
      }
      pendingCityRef.current = null;
    }
  }, [cities, setValue]);

  // Reset dependent fields when country changes
  const prevCountryRef = useRef(country);
  useEffect(() => {
    if (prevCountryRef.current !== country) {
      prevCountryRef.current = country;
      setValue('address.state', '', { shouldValidate: false });
      setValue('address.city', '', { shouldValidate: false });
      setValue('address.zip_code', '', { shouldValidate: false });
      setValue('address.street', '', { shouldValidate: false });
      setValue('address.neighborhood', '', { shouldValidate: false });
      clearCities();
    }
  }, [country, setValue, clearCities]);

  // Clear city when state changes (Brazil only, not from CEP)
  const prevStateRef = useRef(stateValue);
  useEffect(() => {
    if (prevStateRef.current !== stateValue) {
      prevStateRef.current = stateValue;
      if (!pendingCityRef.current) {
        setValue('address.city', '', { shouldValidate: false });
      }
    }
  }, [stateValue, setValue]);

  return { country, stateValue, isBrazil, cepLoading, citiesLoading, cityOptions };
}

export function AddressSection({ form }: AddressSectionProps) {
  const { register, setValue } = form;
  const { country, stateValue, isBrazil, cepLoading, citiesLoading, cityOptions } =
    useAddressAutofill(form);

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-900">Endereço</h3>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <SearchableSelect
          label="País"
          options={countryOptions}
          value={country}
          onChange={(val) => setValue('address.country', val, { shouldValidate: true })}
          placeholder="Buscar país..."
          allowCustom
          customPlaceholder="Digite o país..."
        />
        <div className="relative">
          <Input
            label={isBrazil ? 'CEP' : 'ZIP Code'}
            registration={register('address.zip_code')}
            placeholder={isBrazil ? '00000-000' : ''}
          />
          {isBrazil && cepLoading && (
            <span className="absolute right-3 top-8 text-xs text-blue-600">
              Buscando...
            </span>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Input
          label="Logradouro"
          className="sm:col-span-2"
          registration={register('address.street')}
        />
        <Input label="Número" registration={register('address.number')} />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Input
          label="Complemento"
          registration={register('address.complement')}
        />
        <Input
          label="Bairro"
          registration={register('address.neighborhood')}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <StateCityFields
          form={form}
          isBrazil={isBrazil}
          stateValue={stateValue}
          cityOptions={cityOptions}
          citiesLoading={citiesLoading}
        />
      </div>
    </div>
  );
}
