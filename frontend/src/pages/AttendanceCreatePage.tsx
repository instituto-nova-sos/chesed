import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { useForm, type UseFormRegister, type FieldErrors } from 'react-hook-form';
import { createAttendanceWithOfflineFallback } from '../offline/attendanceOffline';
import { getPerson } from '../api/persons';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
import { useLinkableCampaigns } from '../hooks/useCampaigns';
import { useAuthStore } from '../store/authStore';
import { Button } from '../components/ui/Button';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { Select } from '../components/ui/Select';
import type { CreateAttendanceInput, PersonDetail } from '../types';

interface FormValues {
  service_type_id: string;
  observations: string;
  recommendations: string;
  campaign_id: string;
}

interface AttendanceContext {
  person: PersonDetail | null;
  serviceTypes: ServiceType[];
  loadingContext: boolean;
  loadError: string | null;
}

function useAttendanceContext(personID: string | null): AttendanceContext {
  const [person, setPerson] = useState<PersonDetail | null>(null);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);
  // Start loading only when there is a person to load; with no person_id the
  // effect below short-circuits, so deriving the initial value here avoids a
  // synchronous setState inside the effect (react-hooks/set-state-in-effect).
  const [loadingContext, setLoadingContext] = useState(() => personID != null);
  const [loadError, setLoadError] = useState<string | null>(null);

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
        setLoadError(err instanceof Error ? err.message : 'Falha ao carregar contexto');
      })
      .finally(() => setLoadingContext(false));
  }, [personID]);

  return { person, serviceTypes, loadingContext, loadError };
}

function NoPersonNotice() {
  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Novo Atendimento</h1>
      <Alert variant="info">
        Selecione uma pessoa antes de iniciar um atendimento.{' '}
        <Link to="/persons" className="font-medium underline">
          Ir para pessoas
        </Link>
      </Alert>
    </div>
  );
}

function PersonContextHeader({
  person,
  triageID,
}: {
  person: PersonDetail;
  triageID: string | undefined;
}) {
  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm">
      <span className="text-gray-600">Pessoa: </span>
      <Link to={`/persons/${person.id}`} className="font-medium text-blue-600 hover:underline">
        {person.full_name}
      </Link>
      {triageID && (
        <>
          <span className="text-gray-600"> · Triagem: </span>
          <Link to={`/triages/${triageID}`} className="font-medium text-blue-600 hover:underline">
            ver triagem
          </Link>
        </>
      )}
    </div>
  );
}

interface AttendanceFormProps {
  register: UseFormRegister<FormValues>;
  errors: FieldErrors<FormValues>;
  serviceOptions: { value: string; label: string }[];
  campaignOptions: { value: string; label: string }[];
  submitting: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

function AttendanceForm({
  register,
  errors,
  serviceOptions,
  campaignOptions,
  submitting,
  onSubmit,
  onCancel,
}: AttendanceFormProps) {
  return (
    <form onSubmit={onSubmit} className="max-w-2xl space-y-4">
      <Select
        id="attendance-service-type"
        label="Tipo de Serviço *"
        options={serviceOptions}
        placeholder="Selecione um serviço"
        {...register('service_type_id', { required: 'Selecione um tipo de serviço' })}
        error={errors.service_type_id?.message}
      />

      <Select
        id="attendance-campaign"
        label="Campanha (opcional)"
        options={campaignOptions}
        placeholder="Sem campanha"
        {...register('campaign_id')}
      />

      <div>
        <label className="block text-sm font-medium text-gray-700">Observações</label>
        <textarea
          {...register('observations')}
          rows={3}
          className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700">Recomendações</label>
        <textarea
          {...register('recommendations')}
          rows={3}
          className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div className="flex gap-3">
        <Button type="submit" disabled={submitting}>
          {submitting ? 'Salvando...' : 'Agendar Atendimento'}
        </Button>
        <Button type="button" variant="secondary" onClick={onCancel} disabled={submitting}>
          Cancelar
        </Button>
      </div>
    </form>
  );
}

interface SubmitParams {
  personID: string;
  triageID: string | undefined;
  professionalID: string | null;
  serviceTypes: ServiceType[];
  personName: string | undefined;
}

function useAttendanceSubmit({
  personID,
  triageID,
  professionalID,
  serviceTypes,
  personName,
}: SubmitParams) {
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  async function onSubmit(values: FormValues) {
    if (!professionalID) {
      setSubmitError('Não foi possível identificar o profissional. Complete seu perfil.');
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      const input: CreateAttendanceInput = {
        person_id: personID,
        triage_id: triageID,
        service_type_id: values.service_type_id,
        professional_id: professionalID,
        observations: values.observations || undefined,
        recommendations: values.recommendations || undefined,
      };
      // Assigned conditionally (not `|| undefined`) so the key is absent from
      // the sync-queue payload and the wire body when no campaign is linked.
      if (values.campaign_id) input.campaign_id = values.campaign_id;
      const serviceTypeName = serviceTypes.find(
        (st) => st.id === values.service_type_id,
      )?.name;
      const created = await createAttendanceWithOfflineFallback(
        input,
        personName,
        serviceTypeName,
      );
      // Offline-created records live in the local list until they sync, so we
      // return to the list rather than a detail page the server can't serve yet.
      navigate(created.offline ? '/attendances' : `/attendances/${created.id}`);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Falha ao criar atendimento');
    } finally {
      setSubmitting(false);
    }
  }

  return { submitting, submitError, onSubmit };
}

export function AttendanceCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const personID = searchParams.get('person_id');
  const triageID = searchParams.get('triage_id') ?? undefined;
  const personIDFromAuth = useAuthStore((s) => s.personId);
  const campaigns = useLinkableCampaigns();

  const { person, serviceTypes, loadingContext, loadError } = useAttendanceContext(personID);

  const { register, handleSubmit, formState } = useForm<FormValues>({
    defaultValues: {
      service_type_id: '',
      observations: '',
      recommendations: '',
      campaign_id: '',
    },
  });

  const { submitting, submitError, onSubmit } = useAttendanceSubmit({
    personID: personID ?? '',
    triageID,
    professionalID: personIDFromAuth,
    serviceTypes,
    personName: person?.full_name,
  });

  if (!personID) return <NoPersonNotice />;
  if (loadingContext) return <LoadingScreen />;
  if (!person) {
    return <Alert variant="error">{loadError ?? 'Pessoa não encontrada ou sem permissão.'}</Alert>;
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Novo Atendimento</h1>

      <PersonContextHeader person={person} triageID={triageID} />

      {submitError && <Alert variant="error">{submitError}</Alert>}

      <AttendanceForm
        register={register}
        errors={formState.errors}
        serviceOptions={serviceTypes.map((st) => ({ value: st.id, label: st.name }))}
        campaignOptions={campaigns.map((c) => ({ value: c.id, label: c.name }))}
        submitting={submitting}
        onSubmit={handleSubmit(onSubmit)}
        onCancel={() => navigate(-1)}
      />
    </div>
  );
}
