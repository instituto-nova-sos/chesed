import { useCallback } from 'react';
import type { UseFormReturn } from 'react-hook-form';
import { Input } from '../ui/Input';
import { PhoneInput } from '../ui/PhoneInput';
import { ReferralSourceSelect } from './ReferralSourceSelect';

interface ContactSectionProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  form: UseFormReturn<any, any, any>;
  emailReadOnly?: boolean;
  emailValue?: string;
}

function errMsg(err: unknown): string | undefined {
  if (!err) return undefined;
  if (typeof err === 'object' && 'message' in err) return (err as { message?: string }).message;
  return undefined;
}

export function ContactSection({ form, emailReadOnly, emailValue }: ContactSectionProps) {
  const {
    register,
    setValue,
    watch,
    formState: { errors },
  } = form;

  const phoneValue = watch('phone') || '';

  const handlePhoneChange = useCallback(
    (e164: string) => {
      setValue('phone', e164, { shouldValidate: true });
    },
    [setValue],
  );

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-900">Contato</h3>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {emailReadOnly && emailValue ? (
          <div>
            <label className="block text-sm font-medium text-gray-700">Email</label>
            <input
              type="email"
              value={emailValue}
              readOnly
              className="mt-1 block w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-500 shadow-sm"
            />
          </div>
        ) : (
          <Input
            label="Email"
            type="email"
            error={errMsg(errors.email)}
            registration={register('email')}
            placeholder="email@exemplo.com"
          />
        )}
        <PhoneInput
          label="Telefone"
          value={phoneValue}
          onChange={handlePhoneChange}
          error={errMsg(errors.phone)}
        />
      </div>

      <ReferralSourceSelect
        registration={register('referral_source')}
        error={errMsg(errors.referral_source)}
        defaultValue={watch('referral_source') || ''}
        onValueChange={(val) => setValue('referral_source', val, { shouldValidate: true })}
      />
    </div>
  );
}
