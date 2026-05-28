import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams, Link } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { createTriage } from '../api/triages';
import { getPerson } from '../api/persons';
import { listServiceTypes, type ServiceType } from '../api/serviceTypes';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import type { CreateTriageInput, PersonDetail } from '../types';

interface FormValues {
  main_complaint: string;
  location: string;
  notes: string;
  requested_service_types: string[];
}

export function TriageCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const personID = searchParams.get('person_id');

  const [person, setPerson] = useState<PersonDetail | null>(null);
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>([]);
  const [loadingContext, setLoadingContext] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const { register, handleSubmit, watch, setValue, formState } =
    useForm<FormValues>({
      defaultValues: { main_complaint: '', location: '', notes: '', requested_service_types: [] },
    });
  const selected = watch('requested_service_types');

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

  if (loadingContext) return <LoadingScreen />;
  if (!person) {
    return (
      <Alert variant="error">Pessoa não encontrada ou sem permissão de acesso.</Alert>
    );
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
      const input: CreateTriageInput = {
        person_id: personID!,
        main_complaint: values.main_complaint,
        location: values.location || undefined,
        notes: values.notes || undefined,
        requested_service_types: values.requested_service_types,
      };
      const created = await createTriage(input);
      navigate(`/triages/${created.id}`);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Falha ao criar triagem');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">Nova Triagem</h1>

      <div className="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm">
        <span className="text-gray-600">Pessoa: </span>
        <Link to={`/persons/${person.id}`} className="font-medium text-blue-600 hover:underline">
          {person.full_name}
        </Link>
      </div>

      {submitError && <Alert variant="error">{submitError}</Alert>}

      <form onSubmit={handleSubmit(onSubmit)} className="max-w-2xl space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">
            Queixa Principal *
          </label>
          <textarea
            {...register('main_complaint', { required: 'Queixa é obrigatória', maxLength: 2000 })}
            rows={4}
            className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {formState.errors.main_complaint && (
            <p className="mt-1 text-xs text-red-600">
              {formState.errors.main_complaint.message}
            </p>
          )}
        </div>

        <Input label="Local" {...register('location', { maxLength: 300 })} />

        <div>
          <label className="block text-sm font-medium text-gray-700">
            Serviços Solicitados
          </label>
          <div className="mt-2 flex flex-wrap gap-2">
            {serviceTypes.length === 0 ? (
              <p className="text-xs text-gray-500">Nenhum tipo de serviço disponível.</p>
            ) : (
              serviceTypes.map((st) => {
                const checked = selected.includes(st.id);
                return (
                  <button
                    key={st.id}
                    type="button"
                    onClick={() => toggleService(st.id)}
                    className={`rounded-full px-3 py-1 text-xs font-medium ${
                      checked
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                    }`}
                  >
                    {st.name}
                  </button>
                );
              })
            )}
          </div>
        </div>

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
