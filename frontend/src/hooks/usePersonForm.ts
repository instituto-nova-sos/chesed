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
    defaultValues: {
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
    },
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

  function cleanInput(data: CreatePersonFormData) {
    const cleaned: Record<string, unknown> = { ...data };
    for (const [key, value] of Object.entries(cleaned)) {
      if (value === '' || value === undefined) {
        delete cleaned[key];
      }
    }
    if (data.address) {
      const addr: Record<string, unknown> = { ...data.address };
      let hasValue = false;
      for (const [key, value] of Object.entries(addr)) {
        if (value === '' || value === undefined) {
          delete addr[key];
        } else {
          hasValue = true;
        }
      }
      cleaned.address = hasValue ? addr : undefined;
    }
    return cleaned;
  }

  async function onSubmit(data: CreatePersonFormData) {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const cleaned = cleanInput(data);
      if (editId) {
        await updatePerson(editId, cleaned as CreatePersonFormData);
        navigate(`/persons/${editId}`);
      } else {
        const created = await createPersonWithOfflineFallback(
          cleaned as CreatePersonFormData,
        );
        // Offline-created records live in the local list until they sync, so we
        // return to the list rather than a detail page the server can't serve yet.
        navigate(created.offline ? '/persons' : `/persons/${created.id}`);
      }
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
