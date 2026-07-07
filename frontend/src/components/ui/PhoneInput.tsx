import { useCallback, useState } from 'react';
import { parsePhoneNumberFromString } from 'libphonenumber-js';
import { PHONE_COUNTRY_CODES, getAlpha2ByPhoneCode } from '../../utils/countries';

interface PhoneInputProps {
  value: string;
  onChange: (e164: string) => void;
  error?: string;
  label: string;
}

/**
 * Applies a national phone mask to raw digits based on country.
 * Brazil: (XX) XXXXX-XXXX or (XX) XXXX-XXXX
 * Other countries: digits only (no mask).
 */
function applyNationalMask(digits: string, alpha2?: string): string {
  if (!digits) return '';

  if (alpha2 === 'BR') {
    // Truncate to max 11 digits (2 DDD + 9 subscriber)
    const d = digits.substring(0, 11);
    if (d.length <= 2) {
      return `(${d}`;
    }
    const ddd = d.substring(0, 2);
    const rest = d.substring(2);
    if (rest.length <= 5) {
      return `(${ddd}) ${rest}`;
    }
    // Mobile (9 digits after DDD) or landline (8 digits)
    const splitAt = rest.length > 8 ? 5 : 4;
    return `(${ddd}) ${rest.substring(0, splitAt)}-${rest.substring(splitAt)}`;
  }

  // Other countries: return digits as-is (no specific mask)
  return digits;
}

function parseInitialValue(value: string, fallbackCode: string = '+55'): {
  code: string;
  national: string;
} {
  if (value && value.startsWith('+')) {
    const parsed = parsePhoneNumberFromString(value);
    if (parsed) {
      const code = `+${parsed.countryCallingCode}`;
      const alpha2 = getAlpha2ByPhoneCode(code);
      const national = applyNationalMask(parsed.nationalNumber, alpha2);
      return { code, national };
    }
    // Fallback: strip known country code prefix and mask the remaining digits
    if (value.startsWith(fallbackCode)) {
      const digits = value.substring(fallbackCode.length).replace(/\D/g, '');
      const alpha2 = getAlpha2ByPhoneCode(fallbackCode);
      return { code: fallbackCode, national: applyNationalMask(digits, alpha2) };
    }
  }
  const digits = (value || '').replace(/\D/g, '');
  const alpha2 = getAlpha2ByPhoneCode(fallbackCode);
  return { code: fallbackCode, national: digits ? applyNationalMask(digits, alpha2) : '' };
}

function toE164(code: string, national: string): string {
  const digits = national.replace(/\D/g, '');
  if (digits.length === 0) return '';
  const parsed = parsePhoneNumberFromString(`${code}${digits}`);
  return parsed ? parsed.format('E.164') : `${code}${digits}`;
}

export function PhoneInput({ value, onChange, error, label }: PhoneInputProps) {
  const [initial] = useState(() => parseInitialValue(value));
  const [countryCode, setCountryCode] = useState(initial.code);
  const [nationalNumber, setNationalNumber] = useState(initial.national);

  // Sync internal state when the external value changes (e.g., form reset on
  // edit). Derived during render instead of in an effect. Updates that echo the
  // component's own current E.164 output are skipped so user typing is not
  // reformatted mid-edit.
  const [prevValue, setPrevValue] = useState(value);
  if (prevValue !== value) {
    setPrevValue(value);
    if (value !== toE164(countryCode, nationalNumber)) {
      if (value) {
        const parsed = parseInitialValue(value, countryCode);
        setCountryCode(parsed.code);
        setNationalNumber(parsed.national);
      } else {
        setNationalNumber('');
      }
    }
  }

  const handleNumberChange = useCallback(
    (raw: string) => {
      const digits = raw.replace(/\D/g, '');
      const alpha2 = getAlpha2ByPhoneCode(countryCode);
      setNationalNumber(applyNationalMask(digits, alpha2));
      onChange(toE164(countryCode, digits));
    },
    [countryCode, onChange],
  );

  const handleCountryChange = useCallback(
    (newCode: string) => {
      setCountryCode(newCode);
      const digits = nationalNumber.replace(/\D/g, '');
      if (digits.length > 0) {
        const alpha2 = getAlpha2ByPhoneCode(newCode);
        setNationalNumber(applyNationalMask(digits, alpha2));
        onChange(toE164(newCode, digits));
      }
    },
    [nationalNumber, onChange],
  );

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      <div className="mt-1 flex">
        <select
          value={countryCode}
          onChange={(e) => handleCountryChange(e.target.value)}
          className="rounded-l-lg border border-r-0 border-gray-300 bg-gray-50 px-2 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {PHONE_COUNTRY_CODES.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        <input
          type="tel"
          value={nationalNumber}
          onChange={(e) => handleNumberChange(e.target.value)}
          placeholder="(21) 98765-4321"
          className={`block w-full rounded-r-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
            error
              ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
              : 'border-gray-300 focus:border-blue-500'
          }`}
        />
      </div>
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </div>
  );
}
