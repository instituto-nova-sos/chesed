import { useCallback, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from 'react-router-dom';
import {
  createPersonSchema,
  type CreatePersonFormData,
} from '../utils/personValidation';
import { updatePerson, checkDuplicate } from '../api/persons';
import { createPersonWithOfflineFallback } from '../offline/personOffline';
import type { DuplicateCheckResult } from '../types';

function stripEmpty(source: Record<string, unknown>): {
  cleaned: Record<string, unknown>;
  hasValue: boolean;
} {
  const cleaned: Record<string, unknown> = { ...source };
  let hasValue = false;
  for (const [key, value] of Object.entries(cleaned)) {
    if (value === '' || value === undefined) {
      delete cleaned[key];
    } else {
      hasValue = true;
    }
  }
  return { cleaned, hasValue };
}

function cleanInput(data: CreatePersonFormData): Record<string, unknown> {
  const { cleaned } = stripEmpty(data as Record<string, unknown>);
  if (data.address) {
    const { cleaned: addr, hasValue } = stripEmpty(
      data.address as Record<string, unknown>,
    );
    cleaned.address = hasValue ? addr : undefined;
  }
  return cleaned;
}

const defaultPersonFormValues: CreatePersonFormData = {
  full_name: '',
  document_type: 'CPF',
  document_number: '',
  nationality: 'BRA',
  birth_date: '',
  gender: undefined,
  email: '',
  phone: '',
  referral_source: '',
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
};

async function persistPerson(
  editId: string | undefined,
  cleaned: CreatePersonFormData,
): Promise<{ path: string }> {
  if (editId) {
    await updatePerson(editId, cleaned);
    return { path: `/persons/${editId}` };
  }
  const created = await createPersonWithOfflineFallback(cleaned);
  // Offline-created records live in the local list until they sync, so we
  // return to the list rather than a detail page the server can't serve yet.
  return { path: created.offline ? '/persons' : `/persons/${created.id}` };
}

export function usePersonForm(editId?: string) {
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [duplicateWarning, setDuplicateWarning] =
    useState<DuplicateCheckResult | null>(null);
  const lastCheckedRef = useRef<string>('');

  const form = useForm<CreatePersonFormData>({
    resolver: zodResolver(createPersonSchema),
    mode: 'onBlur',
    defaultValues: defaultPersonFormValues,
  });

  const checkForDuplicates = useCallback(
    async (documentType: string, documentNumber: string) => {
      if (!documentNumber) {
        setDuplicateWarning(null);
        lastCheckedRef.current = '';
        return;
      }
      const key = `${documentType}:${documentNumber}`;
      if (lastCheckedRef.current === key) return;
      try {
        const result = await checkDuplicate(documentType, documentNumber);
        lastCheckedRef.current = key;
        setDuplicateWarning(result.has_duplicates ? result : null);
      } catch {
        setDuplicateWarning(null);
      }
    },
    [],
  );

  function clearDuplicateWarning() {
    setDuplicateWarning(null);
  }

  async function onSubmit(data: CreatePersonFormData) {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const cleaned = cleanInput(data) as CreatePersonFormData;
      const { path } = await persistPerson(editId, cleaned);
      navigate(path);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : 'Erro ao salvar pessoa',
      );
    } finally {
      setIsSubmitting(false);
    }
  }

  return {
    form,
    isSubmitting,
    submitError,
    duplicateWarning,
    checkForDuplicates,
    clearDuplicateWarning,
    onSubmit,
  };
}
