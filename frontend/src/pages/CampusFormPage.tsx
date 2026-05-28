import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import {
  createCampus,
  updateCampus,
  getCampus,
  type CampusInput,
} from '../api/campuses';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Alert } from '../components/ui/Alert';

const REGIONS = [
  { value: 'BRAZIL', label: 'Brasil' },
  { value: 'USA', label: 'Estados Unidos' },
  { value: 'EUROPE', label: 'Europa' },
];

export function CampusFormPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEdit = Boolean(id);

  const [loading, setLoading] = useState(isEdit);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<CampusInput & { is_active: boolean }>({
    defaultValues: {
      name: '',
      region: 'BRAZIL',
      city: '',
      state: '',
      country: 'BRA',
      is_active: true,
    },
  });

  useEffect(() => {
    if (!id) return;
    getCampus(id)
      .then((campus) => {
        form.reset({
          name: campus.name,
          region: campus.region,
          city: campus.city ?? '',
          state: campus.state ?? '',
          country: campus.country,
          is_active: campus.is_active,
        });
      })
      .catch(() => setError('Erro ao carregar campus'))
      .finally(() => setLoading(false));
  }, [id, form]);

  async function onSubmit(data: CampusInput & { is_active: boolean }) {
    setSubmitting(true);
    setError(null);

    const input: CampusInput & { is_active?: boolean } = {
      name: data.name,
      region: data.region,
      city: data.city || undefined,
      state: data.state || undefined,
      country: data.country,
    };

    try {
      if (isEdit && id) {
        await updateCampus(id, { ...input, is_active: data.is_active });
      } else {
        await createCampus(input);
      }
      navigate('/campuses');
    } catch {
      setError('Erro ao salvar campus');
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">
        {isEdit ? 'Editar Campus' : 'Novo Campus'}
      </h1>

      {error && (
        <div className="mb-4">
          <Alert variant="error">{error}</Alert>
        </div>
      )}

      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="max-w-lg space-y-4"
      >
        <Input
          label="Nome *"
          {...form.register('name', { required: 'Nome é obrigatório' })}
          error={form.formState.errors.name?.message}
        />

        <Select
          label="Região *"
          options={REGIONS}
          {...form.register('region', { required: 'Região é obrigatória' })}
          error={form.formState.errors.region?.message}
        />

        <Input label="Cidade" {...form.register('city')} />

        <Input label="Estado" {...form.register('state')} />

        <Input
          label="País (código ISO) *"
          {...form.register('country', {
            required: 'País é obrigatório',
            maxLength: { value: 3, message: 'Máximo 3 caracteres' },
          })}
          error={form.formState.errors.country?.message}
          placeholder="BRA"
        />

        {isEdit && (
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              {...form.register('is_active')}
              className="rounded border-gray-300 text-blue-600"
            />
            <span className="text-sm text-gray-700">Campus ativo</span>
          </label>
        )}

        <div className="flex items-center gap-3 pt-4">
          <Button type="submit" isLoading={submitting}>
            {isEdit ? 'Salvar' : 'Criar Campus'}
          </Button>
          <button
            type="button"
            onClick={() => navigate('/campuses')}
            className="text-sm text-gray-500 hover:text-gray-700"
          >
            Cancelar
          </button>
        </div>
      </form>
    </div>
  );
}
