import { useCallback, useEffect, useMemo, useState } from 'react';
import type { UseFormReturn } from 'react-hook-form';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { SearchableSelect } from '../ui/SearchableSelect';
import { COUNTRIES, getDocumentTypeOptions } from '../../utils/countries';
import { BRAZIL_STATES } from '../../utils/brazilStates';
import { formatCPF } from '../../utils/cpfValidation';

interface PersonalDataSectionProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: UseFormReturn<any, any, any>;
}

const genderOptions = [
  { value: '', label: 'Selecione...' },
  { value: 'M', label: 'Masculino' },
  { value: 'F', label: 'Feminino' },
  { value: 'OTHER', label: 'Outro' },
  { value: 'PREFER_NOT_TO_SAY', label: 'Prefiro não dizer' },
];

const RG_AUTHORITY_OPTIONS = [
  { value: '', label: 'Selecione...' },
  { value: 'SSP', label: 'SSP' },
  { value: 'DETRAN', label: 'DETRAN' },
  { value: 'IFP', label: 'IFP' },
  { value: 'PC', label: 'PC' },
  { value: 'OTHER', label: 'Outro' },
];

const rgStateOptions = [
  { value: '', label: 'UF...' },
  ...BRAZIL_STATES.map((s) => ({ value: s.code, label: `${s.name} (${s.code})` })),
];

const nationalityOptions = COUNTRIES.map((c) => ({
  value: c.code,
  label: `${c.flag} ${c.name}`,
  searchTerms: `${c.alpha2} ${c.code}`,
}));

function errMsg(err: unknown): string | undefined {
  if (!err) return undefined;
  if (typeof err === 'object' && 'message' in err) return (err as { message?: string }).message;
  return undefined;
}

export function PersonalDataSection({ form }: PersonalDataSectionProps) {
  const {
    register,
    watch,
    setValue,
    formState: { errors },
  } = form;

  const nationality = watch('nationality') || 'BRA';
  const documentType = watch('document_type');
  const docNumber = watch('document_number') || '';

  // RG local state for three-field input
  const [rgAuthority, setRgAuthority] = useState('');
  const [rgAuthorityCustom, setRgAuthorityCustom] = useState('');
  const [rgState, setRgState] = useState('');
  const [rgNumber, setRgNumber] = useState('');

  // Parse existing document_number into RG sub-fields on edit
  // Supports both new format "SSP/BA 1234567" and legacy "SSP/BA 1234567"
  useEffect(() => {
    if (documentType === 'RG' && docNumber) {
      const spaceIdx = docNumber.indexOf(' ');
      if (spaceIdx > 0) {
        const authorityPart = docNumber.substring(0, spaceIdx);
        const numberPart = docNumber.substring(spaceIdx + 1);
        const slashIdx = authorityPart.indexOf('/');
        if (slashIdx > 0) {
          const auth = authorityPart.substring(0, slashIdx);
          const state = authorityPart.substring(slashIdx + 1);
          const isKnown = RG_AUTHORITY_OPTIONS.some(
            (o) => o.value === auth && o.value !== '' && o.value !== 'OTHER',
          );
          setRgAuthority(isKnown ? auth : 'OTHER');
          setRgAuthorityCustom(isKnown ? '' : auth);
          setRgState(state);
        } else {
          // Legacy format without slash
          setRgAuthority('OTHER');
          setRgAuthorityCustom(authorityPart);
          setRgState('');
        }
        setRgNumber(numberPart);
      } else {
        setRgNumber(docNumber);
      }
    }
    // Only parse when switching to RG type
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [documentType]);

  const handleDocNumberChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value;
      if (documentType === 'CPF') {
        setValue('document_number', formatCPF(raw), { shouldValidate: true });
      } else {
        setValue('document_number', raw, { shouldValidate: true });
      }
    },
    [documentType, setValue],
  );

  const buildRgValue = useCallback(
    (authority: string, customAuth: string, state: string, number: string) => {
      const effectiveAuth = authority === 'OTHER' ? customAuth : authority;
      if (effectiveAuth && state && number) {
        return `${effectiveAuth}/${state} ${number}`;
      }
      if (effectiveAuth && number) {
        return `${effectiveAuth} ${number}`;
      }
      return number || '';
    },
    [],
  );

  const handleRgFieldChange = useCallback(
    (authority: string, customAuth: string, state: string, number: string) => {
      const combined = buildRgValue(authority, customAuth, state, number);
      setValue('document_number', combined, { shouldValidate: true });
    },
    [setValue, buildRgValue],
  );

  const documentTypeOptions = useMemo(
    () => getDocumentTypeOptions(nationality),
    [nationality],
  );

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-900">Dados Pessoais</h3>

      <Input
        label="Nome Completo *"
        error={errMsg(errors.full_name)}
        registration={register('full_name')}
        placeholder="Ex: Maria da Silva Santos"
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <SearchableSelect
          label="Nacionalidade"
          options={nationalityOptions}
          value={nationality}
          onChange={(val) => setValue('nationality', val, { shouldValidate: true })}
          placeholder="Buscar país..."
          allowCustom
          customPlaceholder="Digite a nacionalidade..."
        />
        <Select
          label="Tipo de Documento *"
          options={documentTypeOptions}
          error={errMsg(errors.document_type)}
          registration={register('document_type')}
        />
      </div>

      {documentType === 'RG' ? (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Select
              label="Órgão Emissor"
              options={RG_AUTHORITY_OPTIONS}
              value={rgAuthority}
              onChange={(e) => {
                const val = e.target.value;
                setRgAuthority(val);
                if (val !== 'OTHER') setRgAuthorityCustom('');
                handleRgFieldChange(val, val === 'OTHER' ? rgAuthorityCustom : '', rgState, rgNumber);
              }}
            />
            {rgAuthority === 'OTHER' && (
              <Input
                label="Órgão (especifique)"
                value={rgAuthorityCustom}
                onChange={(e) => {
                  const val = e.target.value.toUpperCase();
                  setRgAuthorityCustom(val);
                  handleRgFieldChange('OTHER', val, rgState, rgNumber);
                }}
                placeholder="IFP"
              />
            )}
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Select
              label="Estado (UF)"
              options={rgStateOptions}
              value={rgState}
              onChange={(e) => {
                const val = e.target.value;
                setRgState(val);
                handleRgFieldChange(rgAuthority, rgAuthorityCustom, val, rgNumber);
              }}
            />
            <Input
              label="Número do RG"
              value={rgNumber}
              onChange={(e) => {
                const val = e.target.value;
                setRgNumber(val);
                handleRgFieldChange(rgAuthority, rgAuthorityCustom, rgState, val);
              }}
              placeholder="1234567"
              error={errMsg(errors.document_number)}
            />
          </div>
        </div>
      ) : (
        <Input
          label="Número do Documento"
          error={errMsg(errors.document_number)}
          value={docNumber}
          onChange={handleDocNumberChange}
          placeholder={documentType === 'CPF' ? '123.456.789-00' : ''}
        />
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Input
          label="Data de Nascimento"
          type="date"
          error={errMsg(errors.birth_date)}
          registration={register('birth_date')}
        />
        <Select
          label="Gênero"
          options={genderOptions}
          error={errMsg(errors.gender)}
          registration={register('gender')}
        />
      </div>
    </div>
  );
}
