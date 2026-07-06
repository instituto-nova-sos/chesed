import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useForm, type UseFormReturn } from 'react-hook-form';
import { createDonation, getDonation, updateDonation } from '../api/donations';
import { useLinkableCampaigns } from '../hooks/useCampaigns';
import { usePersons } from '../hooks/usePersons';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Alert } from '../components/ui/Alert';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { SearchableSelect } from '../components/ui/SearchableSelect';
import { DONATION_TYPE_LABELS } from '../utils/donationLabels';
import type { CampaignListItem, DonationDetail, DonationInput } from '../types';

const TYPE_OPTIONS = Object.entries(DONATION_TYPE_LABELS).map(([value, label]) => ({
  value,
  label,
}));

const CURRENCY_OPTIONS = [
  { value: 'BRL', label: 'Real (BRL)' },
  { value: 'USD', label: 'Dólar (USD)' },
  { value: 'EUR', label: 'Euro (EUR)' },
];

interface FormValues {
  donation_type: string;
  amount: string;
  currency: string;
  item_description: string;
  donation_date: string;
  notes: string;
}

const EMPTY_FORM: FormValues = {
  donation_type: 'FINANCIAL',
  amount: '',
  currency: 'BRL',
  item_description: '',
  donation_date: '',
  notes: '',
};

function toFormValues(d: DonationDetail): FormValues {
  return {
    donation_type: d.donation_type,
    amount: d.amount !== null ? String(d.amount) : '',
    currency: d.currency || 'BRL',
    item_description: d.item_description ?? '',
    donation_date: d.donation_date ? d.donation_date.slice(0, 10) : '',
    notes: d.notes ?? '',
  };
}

function toPayload(data: FormValues, donorId: string, campaignId: string): DonationInput {
  const isFinancial = data.donation_type === 'FINANCIAL';
  return {
    donation_type: data.donation_type,
    amount: isFinancial && data.amount ? Number(data.amount) : undefined,
    currency: data.currency || undefined,
    item_description: data.item_description || undefined,
    donor_person_id: donorId || undefined,
    campaign_id: campaignId || undefined,
    donation_date: data.donation_date || undefined,
    notes: data.notes || undefined,
  };
}

function FinancialFields({ form }: { form: UseFormReturn<FormValues> }) {
  const { register, formState } = form;
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <Input
        id="donation-amount"
        label="Valor"
        type="number"
        step="0.01"
        min="0"
        error={formState.errors.amount?.message}
        registration={register('amount', {
          validate: (value, values) =>
            values.donation_type !== 'FINANCIAL' ||
            (Boolean(value) && Number(value) > 0) ||
            'Informe um valor maior que zero',
        })}
      />
      <Select
        id="donation-currency"
        label="Moeda"
        options={CURRENCY_OPTIONS}
        registration={register('currency', { required: 'Campo obrigatório' })}
      />
    </div>
  );
}

interface CampaignSelectProps {
  campaigns: CampaignListItem[];
  campaignId: string;
  onCampaignChange: (id: string) => void;
}

function CampaignSelect({ campaigns, campaignId, onCampaignChange }: CampaignSelectProps) {
  return (
    <div>
      <label htmlFor="donation-campaign" className="block text-sm font-medium text-gray-700">
        Campanha (opcional)
      </label>
      <select
        id="donation-campaign"
        className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm"
        value={campaignId}
        onChange={(e) => onCampaignChange(e.target.value)}
      >
        <option value="">Nenhuma</option>
        {campaigns.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
    </div>
  );
}

interface DonationFieldsProps {
  form: UseFormReturn<FormValues>;
  campaigns: CampaignListItem[];
  donorId: string;
  onDonorChange: (id: string) => void;
  onDonorSearch: (text: string) => void;
  donorOptions: { value: string; label: string }[];
  campaignId: string;
  onCampaignChange: (id: string) => void;
}

function DonationFields({
  form,
  donorId,
  onDonorChange,
  onDonorSearch,
  donorOptions,
  campaigns,
  campaignId,
  onCampaignChange,
}: DonationFieldsProps) {
  const { register, formState, watch } = form;
  const donationType = watch('donation_type');
  const requiresItem = donationType === 'GOODS' || donationType === 'SERVICES';

  return (
    <>
      <Select
        id="donation-type"
        label="Tipo"
        options={TYPE_OPTIONS}
        registration={register('donation_type', { required: 'Campo obrigatório' })}
      />
      {donationType === 'FINANCIAL' && <FinancialFields form={form} />}
      <Input
        id="donation-item-description"
        label="Descrição do item"
        error={formState.errors.item_description?.message}
        registration={register('item_description', {
          validate: (value, values) =>
            (values.donation_type !== 'GOODS' && values.donation_type !== 'SERVICES') ||
            Boolean(value) ||
            'Campo obrigatório',
        })}
      />
      {requiresItem && <p className="text-xs text-gray-500">Descreva os bens ou serviços doados.</p>}
      <Input
        id="donation-date"
        label="Data da doação"
        type="date"
        registration={register('donation_date')}
      />
      <SearchableSelect
        label="Doador (opcional)"
        options={donorOptions}
        value={donorId}
        onChange={onDonorChange}
        onSearchTextChange={onDonorSearch}
        placeholder="Buscar pessoa..."
      />
      <CampaignSelect
        campaigns={campaigns}
        campaignId={campaignId}
        onCampaignChange={onCampaignChange}
      />
      <Input id="donation-notes" label="Observações" registration={register('notes')} />
    </>
  );
}

export function DonationFormPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [donorId, setDonorId] = useState('');
  const [campaignId, setCampaignId] = useState('');

  const form = useForm<FormValues>({ defaultValues: EMPTY_FORM });
  const campaigns = useLinkableCampaigns();
  const { persons, search } = usePersons();
  const donorOptions = persons.map((p) => ({ value: p.id, label: p.full_name }));

  useEffect(() => {
    if (!id) return;
    getDonation(id)
      .then((d) => {
        form.reset(toFormValues(d));
        setDonorId(d.donor_person_id ?? '');
        setCampaignId(d.campaign_id ?? '');
      })
      .catch(() => setError('Erro ao carregar doação'))
      .finally(() => setLoading(false));
  }, [id, form]);

  async function onSubmit(data: FormValues) {
    setSubmitting(true);
    setError(null);
    try {
      const payload = toPayload(data, donorId, campaignId);
      if (isEdit && id) {
        await updateDonation(id, payload);
        navigate(`/donations/${id}`);
      } else {
        const created = await createDonation(payload);
        navigate(`/donations/${created.id}`);
      }
    } catch {
      setError('Erro ao salvar doação. Verifique os dados e tente novamente.');
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) return <LoadingScreen />;

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <h1 className="text-lg font-semibold text-gray-900">
        {isEdit ? 'Editar doação' : 'Nova doação'}
      </h1>

      {error && <Alert variant="error">{error}</Alert>}

      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4 rounded-lg border border-gray-200 bg-white p-4"
        noValidate
      >
        <DonationFields
          form={form}
          campaigns={campaigns}
          donorId={donorId}
          onDonorChange={setDonorId}
          onDonorSearch={search}
          donorOptions={donorOptions}
          campaignId={campaignId}
          onCampaignChange={setCampaignId}
        />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={() => navigate('/donations')}>
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
