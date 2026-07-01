import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { createAttendanceWithOfflineFallback } from '../offline/attendanceOffline';
import { getPerson } from '../api/persons';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
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
}

export function AttendanceCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const personID = searchParams.get('person_id');
  const triageID = searchParams.get('triage_id') ?? undefined;
  const personIDFromAuth = useAuthStore((s) => s.personId);

  const [person, setPerson] = useState<PersonDetail | null>(null);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);
  const [loadingContext, setLoadingContext] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const { register, handleSubmit, formState } = useForm<FormValues>({
    defaultValues: { service_type_id: '', observations: '', recommendations: '' },
  });

  useEffect(() => {
    if (!personID) {
      setLoadingContext(false);
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

  if (!personID) {
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

  if (loadingContext) return <LoadingScreen />;
  if (!person) {
    return <Alert variant="error">Pessoa não encontrada ou sem permissão.</Alert>;
  }

  async function onSubmit(values: FormValues) {
    if (!personIDFromAuth) {
      setSubmitError('Não foi possível identificar o profissional. Complete seu perfil.');
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      const input: CreateAttendanceInput = {
        person_id: personID!,
        triage_id: triageID,
        service_type_id: values.service_type_id,
        professional_id: personIDFromAuth,
        observations: values.observations || undefined,
        recommendations: values.recommendations || undefined,
      };
      const serviceTypeName = serviceTypes.find(
        (st) => st.id === values.service_type_id,
      )?.name;
      const created = await createAttendanceWithOfflineFallback(
        input,
        person?.full_name,
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

  const serviceOptions = serviceTypes.map((st) => ({ value: st.id, label: st.name }));

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Novo Atendimento</h1>

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

      {submitError && <Alert variant="error">{submitError}</Alert>}

      <form onSubmit={handleSubmit(onSubmit)} className="max-w-2xl space-y-4">
        <Select
          label="Tipo de Serviço *"
          options={serviceOptions}
          placeholder="Selecione um serviço"
          {...register('service_type_id', { required: 'Selecione um tipo de serviço' })}
          error={formState.errors.service_type_id?.message}
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
          <Button
            type="button"
            variant="secondary"
            onClick={() => navigate(-1)}
            disabled={submitting}
          >
            Cancelar
          </Button>
        </div>
      </form>
    </div>
  );
}
