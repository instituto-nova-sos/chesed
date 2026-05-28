import { parsePhoneNumberFromString } from 'libphonenumber-js';

/**
 * Formats a phone number stored in E.164 format for display.
 * Brazil (+55): +55 (21) 98765-4321
 * Other countries: +<code> <national number>
 */
export function formatPhoneDisplay(phone: string): string {
  if (!phone) return '';
  const parsed = parsePhoneNumberFromString(phone);
  if (!parsed) return phone;

  const code = parsed.countryCallingCode;

  if (String(code) === '55' && parsed.nationalNumber) {
    const digits = String(parsed.nationalNumber).substring(0, 11);
    if (digits.length >= 10) {
      const ddd = digits.substring(0, 2);
      const rest = digits.substring(2);
      const splitAt = rest.length > 8 ? 5 : 4;
      const prefix = rest.substring(0, splitAt);
      const suffix = rest.substring(splitAt);
      return `+55 (${ddd}) ${prefix}-${suffix}`;
    }
    return `+55 ${digits}`;
  }

  return parsed.formatInternational();
}
