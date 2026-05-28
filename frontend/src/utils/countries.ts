export type { Country } from './countryData';
export { COUNTRIES } from './countryData';
import { COUNTRIES, COMMON_PHONE_COUNTRIES } from './countryData';

/** Phone country code options for the PhoneInput dropdown (common subset) */
export const PHONE_COUNTRY_CODES = COUNTRIES
  .filter((c) => COMMON_PHONE_COUNTRIES.includes(c.alpha2))
  .map((c) => ({
    value: c.phoneCode,
    label: `${c.flag} ${c.phoneCode}`,
    country: c.code,
    alpha2: c.alpha2,
  }));

/** Look up ISO alpha-2 code by phone calling code (for libphonenumber-js) */
export function getAlpha2ByPhoneCode(phoneCode: string): string | undefined {
  const match = PHONE_COUNTRY_CODES.find((c) => c.value === phoneCode);
  if (match) return match.alpha2;
  return COUNTRIES.find((c) => c.phoneCode === phoneCode)?.alpha2;
}

/**
 * Returns document type options based on nationality.
 * BRA → CPF/RG/OTHER. USA → SSN/OTHER. EU → EU_ID/OTHER. Foreign → PASSPORT/OTHER.
 */
export function getDocumentTypeOptions(
  nationality: string,
  campusCountry: string = 'BRA',
): Array<{ value: string; label: string }> {
  const isForeign = nationality !== campusCountry;
  const options: Array<{ value: string; label: string }> = [];

  switch (nationality) {
    case 'BRA':
      options.push({ value: 'CPF', label: 'CPF' });
      options.push({ value: 'RG', label: 'RG' });
      break;
    case 'USA':
      options.push({ value: 'SSN', label: 'SSN' });
      break;
    case 'PRT':
    case 'ESP':
    case 'FRA':
    case 'DEU':
    case 'ITA':
      options.push({ value: 'EU_ID', label: 'EU ID' });
      break;
    default:
      break;
  }

  if (isForeign) {
    options.push({ value: 'PASSPORT', label: 'Passaporte' });
  }

  options.push({ value: 'OTHER', label: 'Outro' });

  return options;
}
