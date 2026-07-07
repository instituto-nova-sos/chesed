import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import type { UseFormRegister, UseFormHandleSubmit, FieldErrors } from 'react-hook-form';
import { useForm, useWatch } from 'react-hook-form';
import { createTriageWithOfflineFallback } from '../offline/triageOffline';
import { getPerson } from '../api/persons';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
import { useLinkableCampaigns } from '../hooks/useCampaigns';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Select } from '../components/ui/Select';
import type { CreateTriageInput, PersonDetail } from '../types';

interface FormValues {
  main_complaint: string;
  location: string;
  notes: string;
  requested_service_types: string[];
  campaign_id: string;
}

export function TriageCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const personID = searchParams.get('person_id');

  const [person, setPerson] = useState<PersonDetail | null>(null);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);
  // Start loading only when there is a person to load; with no person_id the
  // effect below short-circuits, so deriving the initial value here avoids a
  // synchronous setState inside the effect (react-hooks/set-state-in-effect).
  const [loadingContext, setLoadingContext] = useState(() => personID != null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const campaigns = useLinkableCampaigns();

  const { register, handleSubmit, control, setValue, formState } =
    useForm<FormValues>({
      defaultValues: {
        main_complaint: '',
        location: '',
        notes: '',
        requested_service_types: [],
        campaign_id: '',
      },
    });
  // useWatch (not watch()) so the subscription is memoization-safe when it feeds
  // the effect below — avoids react-hooks/incompatible-library.
  const selected = useWatch({ control, name: 'requested_service_types' });

  useEffect(() => {
    if (!personID) {
      return;
    }
    Promise.all([getPerson(personID), listServiceTypes()])
      .then(([p, s]) => {
        setPerson(p);
        setServiceTypes(s.data.filter((st) => st.is_active));
      })
      .catch((err: unknown) => {
        setSubmitError(err instanceof Error ? err.message : 'Falha ao carregar contexto');
      })
      .finally(() => setLoadingContext(false));
  }, [personID]);

  if (!personID) return <NoPersonSelected />;
  if (loadingContext) return <LoadingScreen />;
  if (!person) {
    return <Alert variant="error">Pessoa não encontrada ou sem permissão de acesso.</Alert>;
  }

  function toggleService(id: string) {
    const current = new Set(selected);
    if (current.has(id)) current.delete(id);
    else current.add(id);
    setValue('requested_service_types', Array.from(current), { shouldDirty: true });
  }

  async function onSubmit(values: FormValues) {
    setSubmitting(true);
    setSubmitError(null);
    try {
      const input = buildTriageInput(values, personID!);
      const created = await createTriageWithOfflineFallback(input, person?.full_name);
      // Offline-created records live in the local list until they sync, so we
      // return to the list rather than a detail page the server can't serve yet.
      navigate(created.offline ? '/triages' : `/triages/${created.id}`);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Falha ao criar triagem');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Nova Triagem</h1>

      <PersonContextBanner person={person} />

      {submitError && <Alert variant="error">{submitError}</Alert>}

      <TriageForm
        register={register}
        errors={formState.errors}
        onSubmit={handleSubmit(onSubmit)}
        serviceTypes={serviceTypes}
        selected={selected}
        onToggleService={toggleService}
        campaignOptions={campaigns.map((c) => ({ value: c.id, label: c.name }))}
        submitting={submitting}
        onCancel={() => navigate(-1)}
      />
    </div>
  );
}

function buildTriageInput(values: FormValues, personID: string): CreateTriageInput {
  const input: CreateTriageInput = {
    person_id: personID,
    main_complaint: values.main_complaint,
    location: values.location || undefined,
    notes: values.notes || undefined,
    requested_service_types: values.requested_service_types,
  };
  // Assigned conditionally (not `|| undefined`) so the key is absent from the
  // sync-queue payload and the wire body when no campaign is linked.
  if (values.campaign_id) input.campaign_id = values.campaign_id;
  return input;
}

function PersonContextBanner({ person }: { person: PersonDetail }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm">
      <span className="text-gray-600">Pessoa: </span>
      <Link to={`/persons/${person.id}`} className="font-medium text-blue-600 hover:underline">
        {person.full_name}
      </Link>
    </div>
  );
}

function NoPersonSelected() {
  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Nova Triagem</h1>
      <Alert variant="info">
        Selecione uma pessoa antes de iniciar uma triagem.{' '}
        <Link to="/persons" className="font-medium underline">
          Ir para pessoas
        </Link>
      </Alert>
    </div>
  );
}

interface TriageFormProps {
  register: UseFormRegister<FormValues>;
  errors: FieldErrors<FormValues>;
  onSubmit: ReturnType<UseFormHandleSubmit<FormValues>>;
  serviceTypes: ServiceType[];
  selected: string[];
  onToggleService: (id: string) => void;
  campaignOptions: { value: string; label: string }[];
  submitting: boolean;
  onCancel: () => void;
}

function TriageForm({
  register,
  errors,
  onSubmit,
  serviceTypes,
  selected,
  onToggleService,
  campaignOptions,
  submitting,
  onCancel,
}: TriageFormProps) {
  return (
    <form onSubmit={onSubmit} className="max-w-2xl space-y-4">
      <div>
        <label htmlFor="triage-main-complaint" className="block text-sm font-medium text-gray-700">
          Queixa Principal *
        </label>
        <textarea
          id="triage-main-complaint"
          {...register('main_complaint', { required: 'Queixa é obrigatória', maxLength: 2000 })}
          rows={4}
          className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        {errors.main_complaint && (
          <p className="mt-1 text-xs text-red-600">{errors.main_complaint.message}</p>
        )}
      </div>

      <Input label="Local" {...register('location', { maxLength: 300 })} />

      <div>
        <label className="block text-sm font-medium text-gray-700">Serviços Solicitados</label>
        <div className="mt-2 flex flex-wrap gap-2">
          {serviceTypes.length === 0 ? (
            <p className="text-xs text-gray-500">Nenhum tipo de serviço disponível.</p>
          ) : (
            serviceTypes.map((st) => (
              <button
                key={st.id}
                type="button"
                onClick={() => onToggleService(st.id)}
                className={`rounded-full px-3 py-1 text-xs font-medium ${
                  selected.includes(st.id)
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                {st.name}
              </button>
            ))
          )}
        </div>
      </div>

      <Select
        id="triage-campaign"
        label="Campanha (opcional)"
        options={campaignOptions}
        placeholder="Sem campanha"
        {...register('campaign_id')}
      />

      <div>
        <label className="block text-sm font-medium text-gray-700">Observações</label>
        <textarea
          {...register('notes')}
          rows={3}
          className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div className="flex gap-3">
        <Button type="submit" disabled={submitting}>
          {submitting ? 'Salvando...' : 'Salvar Triagem'}
        </Button>
        <Button type="button" variant="secondary" onClick={onCancel} disabled={submitting}>
          Cancelar
        </Button>
      </div>
    </form>
  );
}
