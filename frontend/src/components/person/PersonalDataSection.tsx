import { useMemo, useState } from 'react';
import type { UseFormReturn } from 'react-hook-form';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';
import { SearchableSelect } from '../ui/SearchableSelect';
import { COUNTRIES, getDocumentTypeOptions } from '../../utils/countries';
import { BRAZIL_STATES } from '../../utils/brazilStates';
import { formatCPF } from '../../utils/cpfValidation';
import { documentNumberPlaceholder } from '../../utils/documentFormat';

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

interface RgFields {
  authority: string;
  authorityCustom: string;
  state: string;
  number: string;
}

const EMPTY_RG: RgFields = { authority: '', authorityCustom: '', state: '', number: '' };

// Parses a stored RG document_number into its sub-fields.
// Supports the "SSP/BA 1234567" format and the legacy "SSP 1234567" (no slash).
function parseRgValue(docNumber: string): RgFields {
  const spaceIdx = docNumber.indexOf(' ');
  if (spaceIdx <= 0) {
    return { ...EMPTY_RG, number: docNumber };
  }
  const authorityPart = docNumber.substring(0, spaceIdx);
  const number = docNumber.substring(spaceIdx + 1);
  const slashIdx = authorityPart.indexOf('/');
  if (slashIdx <= 0) {
    return { authority: 'OTHER', authorityCustom: authorityPart, state: '', number };
  }
  const auth = authorityPart.substring(0, slashIdx);
  const state = authorityPart.substring(slashIdx + 1);
  const isKnown = RG_AUTHORITY_OPTIONS.some(
    (o) => o.value === auth && o.value !== '' && o.value !== 'OTHER',
  );
  return {
    authority: isKnown ? auth : 'OTHER',
    authorityCustom: isKnown ? '' : auth,
    state,
    number,
  };
}

function buildRgValue({ authority, authorityCustom, state, number }: RgFields): string {
  const effectiveAuth = authority === 'OTHER' ? authorityCustom : authority;
  if (effectiveAuth && state && number) return `${effectiveAuth}/${state} ${number}`;
  if (effectiveAuth && number) return `${effectiveAuth} ${number}`;
  return number || '';
}

interface RgDocumentFieldsProps {
  initialValue: string;
  error?: string;
  onChange: (combined: string) => void;
}

function RgDocumentFields({ initialValue, error, onChange }: RgDocumentFieldsProps) {
  const [rg, setRg] = useState<RgFields>(() =>
    initialValue ? parseRgValue(initialValue) : EMPTY_RG,
  );

  const update = (patch: Partial<RgFields>) => {
    const next = { ...rg, ...patch };
    setRg(next);
    onChange(buildRgValue(next));
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Select
          label="Órgão Emissor"
          options={RG_AUTHORITY_OPTIONS}
          value={rg.authority}
          onChange={(e) => {
            const authority = e.target.value;
            update({
              authority,
              authorityCustom: authority === 'OTHER' ? rg.authorityCustom : '',
            });
          }}
        />
        {rg.authority === 'OTHER' && (
          <Input
            label="Órgão (especifique)"
            value={rg.authorityCustom}
            onChange={(e) => update({ authorityCustom: e.target.value.toUpperCase() })}
            placeholder="IFP"
          />
        )}
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Select
          label="Estado (UF)"
          options={rgStateOptions}
          value={rg.state}
          onChange={(e) => update({ state: e.target.value })}
        />
        <Input
          label="Número do RG"
          value={rg.number}
          onChange={(e) => update({ number: e.target.value })}
          placeholder="1234567"
          error={error}
        />
      </div>
    </div>
  );
}

interface DocumentNumberFieldProps {
  documentType: string;
  docNumber: string;
  error?: string;
  setValue: UseFormReturn['setValue'];
}

function DocumentNumberField({ documentType, docNumber, error, setValue }: DocumentNumberFieldProps) {
  const setDocNumber = (value: string) =>
    setValue('document_number', value, { shouldValidate: true });

  if (documentType === 'RG') {
    return (
      <RgDocumentFields initialValue={docNumber} error={error} onChange={setDocNumber} />
    );
  }
  return (
    <Input
      label="Número do Documento"
      error={error}
      value={docNumber}
      onChange={(e) =>
        setDocNumber(documentType === 'CPF' ? formatCPF(e.target.value) : e.target.value)
      }
      placeholder={documentNumberPlaceholder(documentType)}
    />
  );
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

      <DocumentNumberField
        documentType={documentType}
        docNumber={docNumber}
        error={errMsg(errors.document_number)}
        setValue={setValue}
      />

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
