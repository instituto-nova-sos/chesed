/**
 * Offline Behavior: Category A — Fully Offline-Capable
 * - Dexie table: persons
 * - Sync: Create queued when offline
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useEffect } from 'react';
import { Button } from '../ui/Button';
import { Alert } from '../ui/Alert';
import { DuplicateWarning } from './DuplicateWarning';
import { PersonalDataSection } from './PersonalDataSection';
import { ContactSection } from './ContactSection';
import { AddressSection } from './AddressSection';
import { usePersonForm } from '../../hooks/usePersonForm';
import { isValidCPF } from '../../utils/cpfValidation';
import type { PersonDetail } from '../../types';

interface PersonFormProps {
  editData?: PersonDetail;
  emailReadOnly?: boolean;
}

export function PersonForm({ editData, emailReadOnly }: PersonFormProps) {
  const {
    form,
    isSubmitting,
    submitError,
    duplicateWarning,
    checkForDuplicates,
    clearDuplicateWarning,
    onSubmit,
  } = usePersonForm(editData?.id);

  const { handleSubmit, reset, watch } = form;

  useEffect(() => {
    if (editData) {
      reset({
        full_name: editData.full_name,
        document_type: editData.document_type as
          | 'CPF'
          | 'RG'
          | 'SSN'
          | 'EU_ID'
          | 'PASSPORT'
          | 'OTHER',
        document_number: editData.document_number || '',
        nationality: editData.nationality || 'BRA',
        birth_date: editData.birth_date?.split('T')[0] || '',
        gender:
          (editData.gender as 'M' | 'F' | 'OTHER' | 'PREFER_NOT_TO_SAY') ||
          undefined,
        email: editData.email || '',
        phone: editData.phone || '',
        referral_source: editData.referral_source || '',
        address: editData.addresses?.[0]
          ? {
              zip_code: editData.addresses[0].zip_code || '',
              street: editData.addresses[0].street || '',
              number: editData.addresses[0].number || '',
              complement: editData.addresses[0].complement || '',
              neighborhood: editData.addresses[0].neighborhood || '',
              city: editData.addresses[0].city || '',
              state: editData.addresses[0].state || '',
              country: editData.addresses[0].country || 'BRA',
            }
          : undefined,
      });
    }
  }, [editData, reset]);

  const documentType = watch('document_type');
  const documentNumber = watch('document_number');
  const nationality = watch('nationality');

  useEffect(() => {
    if (editData || !documentNumber || !documentType) return undefined;

    if (documentType === 'CPF' && nationality === 'BRA') {
      if (!isValidCPF(documentNumber)) return undefined;
    }

    const timer = setTimeout(() => {
      checkForDuplicates(documentType, documentNumber);
    }, 500);
    return () => clearTimeout(timer);
  }, [documentType, documentNumber, nationality, checkForDuplicates, editData]);

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {submitError && <Alert variant="error">{submitError}</Alert>}

      {duplicateWarning && (
        <DuplicateWarning
          result={duplicateWarning}
          onDismiss={clearDuplicateWarning}
        />
      )}

      <PersonalDataSection form={form} />
      <ContactSection form={form} emailReadOnly={emailReadOnly} />
      <AddressSection form={form} />

      <div className="flex justify-end gap-3 border-t pt-4">
        <Button type="submit" isLoading={isSubmitting}>
          {editData ? 'Salvar Alterações' : 'Cadastrar Pessoa'}
        </Button>
      </div>
    </form>
  );
}
