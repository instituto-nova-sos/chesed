/**
 * Offline Behavior: Category C — Online-Only
 * - Requires network to register and get Keycloak token
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  selfRegisterSchema,
  type SelfRegisterFormData,
} from '../utils/personValidation';
import { selfRegister } from '../api/persons';
import { listActiveCampuses, type Campus } from '../api/campuses';
import { useAuth } from '../hooks/useAuth';
import { PersonalDataSection } from '../components/person/PersonalDataSection';
import { ContactSection } from '../components/person/ContactSection';
import { AddressSection } from '../components/person/AddressSection';
import { Button } from '../components/ui/Button';
import { Alert } from '../components/ui/Alert';
import { keycloak } from '../auth/keycloak';

export function ProfileCompletionPage() {
  const { email, logout } = useAuth();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [campuses, setCampuses] = useState<Campus[]>([]);

  const form = useForm<SelfRegisterFormData>({
    resolver: zodResolver(selfRegisterSchema),
    mode: 'onBlur',
    defaultValues: {
      full_name: '',
      document_type: 'CPF',
      document_number: '',
      nationality: 'BRA',
      birth_date: '',
      gender: undefined,
      phone: '',
      referral_source: '',
      role_type: 'VOLUNTEER',
      campus_id: '',
      address: {
        zip_code: '',
        street: '',
        number: '',
        complement: '',
        neighborhood: '',
        city: '',
        state: '',
        country: 'BRA',
      },
    },
  });

  useEffect(() => {
    listActiveCampuses()
      .then(({ data }) => {
        setCampuses(data);
        if (data.length === 1) {
          form.setValue('campus_id', data[0].id);
        }
      })
      .catch(() => setCampuses([]));
  }, [form]);

  async function onSubmit(data: SelfRegisterFormData) {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const cleaned: Record<string, unknown> = { ...data };
      for (const [key, value] of Object.entries(cleaned)) {
        if (value === '' || value === undefined) delete cleaned[key];
      }
      await selfRegister(cleaned as SelfRegisterFormData);
      setSuccess(true);
      // Force re-login so new claims (person_id, campus_id) are in the token
      setTimeout(() => {
        keycloak.logout({ redirectUri: window.location.origin });
      }, 2000);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : 'Erro ao realizar cadastro',
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
        <div className="w-full max-w-md text-center">
          <Alert variant="success">
            <p className="font-medium">Cadastro realizado com sucesso!</p>
            <p className="mt-1 text-sm">Redirecionando para login...</p>
          </Alert>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
      <div className="w-full max-w-2xl rounded-lg border border-gray-200 bg-white p-6 shadow-lg">
        <div className="mb-6 text-center">
          <h1 className="text-xl font-semibold text-gray-900">
            Complete seu Cadastro
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Para continuar, preencha seus dados e escolha sua função
          </p>
        </div>

        {submitError && (
          <div className="mb-4">
            <Alert variant="error">{submitError}</Alert>
          </div>
        )}

        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {campuses.length > 1 && (
            <div className="space-y-2">
              <label
                htmlFor="campus_id"
                className="text-sm font-medium text-gray-900"
              >
                Campus *
              </label>
              <select
                id="campus_id"
                {...form.register('campus_id')}
                className="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-blue-500"
              >
                <option value="">Selecione um campus</option>
                {campuses.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                    {c.city ? ` - ${c.city}` : ''}
                    {c.state ? `, ${c.state}` : ''}
                  </option>
                ))}
              </select>
              {form.formState.errors.campus_id && (
                <p className="text-xs text-red-600">
                  {form.formState.errors.campus_id.message}
                </p>
              )}
            </div>
          )}

          <PersonalDataSection form={form} />
          <ContactSection form={form} emailReadOnly emailValue={email} />
          <AddressSection form={form} />

          <div className="space-y-4 border-t pt-4">
            <h3 className="text-sm font-medium text-gray-900">
              Função na Organização *
            </h3>
            <div className="flex gap-4">
              <label className="flex items-center gap-2 rounded-lg border border-gray-200 px-4 py-3 cursor-pointer hover:bg-gray-50">
                <input
                  type="radio"
                  value="VOLUNTEER"
                  {...form.register('role_type')}
                  className="text-blue-600"
                />
                <span className="text-sm font-medium">Voluntário</span>
              </label>
              <label className="flex items-center gap-2 rounded-lg border border-gray-200 px-4 py-3 cursor-pointer hover:bg-gray-50">
                <input
                  type="radio"
                  value="ASSISTED"
                  {...form.register('role_type')}
                  className="text-blue-600"
                />
                <span className="text-sm font-medium">Assistido</span>
              </label>
            </div>
            {form.formState.errors.role_type && (
              <p className="text-xs text-red-600">
                {form.formState.errors.role_type.message}
              </p>
            )}
          </div>

          <div className="flex items-center justify-between border-t pt-4">
            <button
              type="button"
              onClick={logout}
              className="text-sm text-gray-500 hover:text-gray-700"
            >
              Sair
            </button>
            <Button type="submit" isLoading={isSubmitting}>
              Cadastrar
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
