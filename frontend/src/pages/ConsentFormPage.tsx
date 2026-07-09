import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useForm, type UseFormReturn } from 'react-hook-form';
import { createConsent } from '../api/consents';
import { ApiError } from '../api/client';
import { useOfflineStatus } from '../hooks/useOfflineStatus';
import { usePersons } from '../hooks/usePersons';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Alert } from '../components/ui/Alert';
import { SearchableSelect } from '../components/ui/SearchableSelect';
import { SignaturePadCanvas } from '../components/ui/SignaturePadCanvas';
import { CONSENT_TYPE_LABELS, PURPOSE_PRESETS } from '../utils/consentLabels';
import type { ConsentType, CreateConsentInput } from '../types/consent';

const TYPE_OPTIONS = Object.entries(CONSENT_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}));

interface FormValues {
  consent_type: ConsentType;
  purpose: string;
  consent_version: string;
}

const EMPTY_FORM: FormValues = {
  consent_type: 'DATA_PROCESSING',
  purpose: PURPOSE_PRESETS.DATA_PROCESSING,
  consent_version: '1.0',
};

function consentErrorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 409) {
    return 'Já existe um consentimento ativo deste tipo.';
  }
  return 'Erro ao registrar consentimento. Tente novamente.';
}

function toPayload(
  data: FormValues,
  personId: string,
  signature: string,
  guardianId: string,
): CreateConsentInput {
  const payload: CreateConsentInput = {
    person_id: personId,
    consent_type: data.consent_type,
    purpose: data.purpose,
    consent_version: data.consent_version,
    signature_data: signature,
  };
  if (guardianId) payload.granted_by_person_id = guardianId;
  return payload;
}

function GuardianPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (id: string) => void;
}) {
  const { persons, search } = usePersons();
  return (
    <SearchableSelect
      label="Responsável (opcional)"
      options={persons.map((p) => ({ value: p.id, label: p.full_name }))}
      value={value}
      onChange={onChange}
      onSearchTextChange={search}
    />
  );
}

interface ConsentFieldsProps {
  form: UseFormReturn<FormValues>;
  guardianId: string;
  onGuardianChange: (id: string) => void;
}

function ConsentFields({ form, guardianId, onGuardianChange }: ConsentFieldsProps) {
  const { register, formState, setValue, watch } = form;
  const consentType = watch('consent_type');

  return (
    <>
      <Select
        id="consent-type"
        label="Tipo de consentimento"
        options={TYPE_OPTIONS}
        registration={register('consent_type', {
          required: 'Campo obrigatório',
          onChange: (event: { target: { value: string } }) => {
            const type = event.target.value as ConsentType;
            setValue('purpose', PURPOSE_PRESETS[type]);
            if (type !== 'MINOR_GUARDIAN') onGuardianChange('');
          },
        })}
      />
      <div>
        <label htmlFor="consent-purpose" className="block text-sm font-medium text-gray-700">
          Finalidade
        </label>
        <textarea
          id="consent-purpose"
          rows={3}
          className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
            formState.errors.purpose
              ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
              : 'border-gray-300 focus:border-blue-500'
          }`}
          {...register('purpose', { required: 'Campo obrigatório' })}
        />
        {formState.errors.purpose && (
          <p className="mt-1 text-xs text-red-600">{formState.errors.purpose.message}</p>
        )}
      </div>
      <Input
        id="consent-version"
        label="Versão"
        readOnly
        registration={register('consent_version')}
      />
      {consentType === 'MINOR_GUARDIAN' && (
        <GuardianPicker value={guardianId} onChange={onGuardianChange} />
      )}
    </>
  );
}

export function ConsentFormPage() {
  const { personId } = useParams<{ personId: string }>();
  const navigate = useNavigate();
  const { isOffline } = useOfflineStatus();

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [signature, setSignature] = useState<string | null>(null);
  const [signatureError, setSignatureError] = useState<string | null>(null);
  const [guardianId, setGuardianId] = useState('');

  const form = useForm<FormValues>({ defaultValues: EMPTY_FORM });

  function handleSignatureChange(dataUrl: string | null) {
    setSignature(dataUrl);
    if (dataUrl) setSignatureError(null);
  }

  async function onSubmit(data: FormValues) {
    if (!personId) return;
    if (!signature) {
      setSignatureError('Assinatura é obrigatória');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createConsent(toPayload(data, personId, signature, guardianId));
      navigate(`/persons/${personId}`);
    } catch (err: unknown) {
      setError(consentErrorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Novo consentimento</h1>

      {isOffline && <Alert variant="warning">Conecte-se para registrar consentimentos.</Alert>}
      {error && <Alert variant="error">{error}</Alert>}

      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4 rounded-lg border border-gray-200 bg-white p-4"
        noValidate
      >
        <fieldset disabled={isOffline} className="space-y-4">
          <ConsentFields form={form} guardianId={guardianId} onGuardianChange={setGuardianId} />
          <div>
            <SignaturePadCanvas
              value={signature ?? undefined}
              onChange={handleSignatureChange}
              disabled={isOffline}
            />
            {signatureError && <p className="mt-1 text-xs text-red-600">{signatureError}</p>}
          </div>
        </fieldset>
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="secondary"
            onClick={() => navigate(`/persons/${personId}`)}
          >
            Cancelar
          </Button>
          <Button type="submit" disabled={submitting || isOffline}>
            {submitting ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </form>
    </div>
  );
}
