import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useForm, type UseFormReturn } from 'react-hook-form';
import { createCampaign, getCampaign, updateCampaign } from '../api/campaigns';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { CAMPAIGN_STATUS_LABELS, CAMPAIGN_TYPE_LABELS } from '../utils/campaignLabels';
import type { Campaign, UpdateCampaignInput } from '../types';

const TYPE_OPTIONS = Object.entries(CAMPAIGN_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}));
const STATUS_OPTIONS = Object.entries(CAMPAIGN_STATUS_LABELS).map(([value, label]) => ({
  value,
  label,
}));

interface FormValues {
  name: string;
  description: string;
  campaign_type: string;
  start_date: string;
  end_date: string;
  location_name: string;
  location_address: string;
  status: string;
}

const EMPTY_FORM: FormValues = {
  name: '',
  description: '',
  campaign_type: 'SOCIAL_ACTION',
  start_date: '',
  end_date: '',
  location_name: '',
  location_address: '',
  status: 'PLANNED',
};

function toFormValues(c: Campaign): FormValues {
  return {
    name: c.name,
    description: c.description ?? '',
    campaign_type: c.campaign_type,
    start_date: c.start_date.slice(0, 10),
    end_date: c.end_date ? c.end_date.slice(0, 10) : '',
    location_name: c.location_name ?? '',
    location_address: c.location_address ?? '',
    status: c.status,
  };
}

function toPayload(data: FormValues): Omit<UpdateCampaignInput, 'status'> {
  return {
    name: data.name,
    campaign_type: data.campaign_type,
    start_date: data.start_date,
    end_date: data.end_date || undefined,
    description: data.description || undefined,
    location_name: data.location_name || undefined,
    location_address: data.location_address || undefined,
  };
}

function CampaignFields({ form, isEdit }: { form: UseFormReturn<FormValues>; isEdit: boolean }) {
  const { register, formState } = form;
  return (
    <>
      <Input
        id="campaign-name"
        label="Nome"
        error={formState.errors.name?.message}
        registration={register('name', { required: 'Campo obrigatório' })}
      />
      <Select
        id="campaign-type"
        label="Tipo"
        options={TYPE_OPTIONS}
        registration={register('campaign_type', { required: 'Campo obrigatório' })}
      />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Input
          id="campaign-start-date"
          label="Data de início"
          type="date"
          error={formState.errors.start_date?.message}
          registration={register('start_date', { required: 'Campo obrigatório' })}
        />
        <Input
          id="campaign-end-date"
          label="Data de término"
          type="date"
          registration={register('end_date')}
        />
      </div>
      <Input id="campaign-location-name" label="Local" registration={register('location_name')} />
      <Input
        id="campaign-location-address"
        label="Endereço do local"
        registration={register('location_address')}
      />
      <Input id="campaign-description" label="Descrição" registration={register('description')} />
      {isEdit && (
        <Select
          id="campaign-status"
          label="Status"
          options={STATUS_OPTIONS}
          registration={register('status')}
        />
      )}
    </>
  );
}

export function CampaignFormPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<FormValues>({ defaultValues: EMPTY_FORM });

  useEffect(() => {
    if (!id) return;
    getCampaign(id)
      .then((c) => form.reset(toFormValues(c)))
      .catch(() => setError('Erro ao carregar campanha'))
      .finally(() => setLoading(false));
  }, [id, form]);

  async function onSubmit(data: FormValues) {
    setSubmitting(true);
    setError(null);
    try {
      if (isEdit && id) {
        await updateCampaign(id, { ...toPayload(data), status: data.status });
        navigate(`/campaigns/${id}`);
      } else {
        const created = await createCampaign(toPayload(data));
        navigate(`/campaigns/${created.id}`);
      }
    } catch {
      setError('Erro ao salvar campanha. Verifique os dados e tente novamente.');
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) return <LoadingScreen />;

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">
        {isEdit ? 'Editar campanha' : 'Nova campanha'}
      </h1>

      {error && <Alert variant="error">{error}</Alert>}

      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4 rounded-lg border border-gray-200 bg-white p-4"
        noValidate
      >
        <CampaignFields form={form} isEdit={isEdit} />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={() => navigate('/campaigns')}>
            Cancelar
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </form>
    </div>
  );
}
