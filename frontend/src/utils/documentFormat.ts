import { isValidCPF } from './cpfValidation';

const SSN_PATTERN = /^\d{3}-?\d{2}-?\d{4}$/;
const GENERIC_DOCUMENT_PATTERN = /^[A-Za-z0-9./\- ]+$/;

/**
 * Validates a US Social Security Number.
 * Accepts formatted (123-45-6789) or unformatted (123456789) input.
 */
export function isValidSSN(value: string): boolean {
  return SSN_PATTERN.test(value);
}

/**
 * Validates a generic identity document number: alphanumeric plus the
 * separators `.`, `-`, `/` and spaces, within the given length bounds.
 */
export function isValidGenericDocument(
  value: string,
  minLength = 3,
  maxLength = 30,
): boolean {
  if (value.length < minLength || value.length > maxLength) return false;
  return GENERIC_DOCUMENT_PATTERN.test(value);
}

/**
 * Portuguese (pt-BR) placeholder/mask hint for the document number input,
 * chosen by document type. Used by the person form UI.
 */
export function documentNumberPlaceholder(docType: string): string {
  switch (docType) {
    case 'CPF':
      return '000.000.000-00';
    case 'SSN':
      return '000-00-0000';
    case 'PASSPORT':
      return 'AB123456';
    default:
      return 'Número do documento';
  }
}

/**
 * Dispatches format validation for an identity document by its type.
 * An empty number is always valid because the field is optional.
 */
export function isValidDocumentFormat(docType: string, number: string): boolean {
  if (number.length === 0) return true;

  switch (docType) {
    case 'CPF':
      return isValidCPF(number);
    case 'SSN':
      return isValidSSN(number);
    case 'RG':
    case 'EU_ID':
    case 'PASSPORT':
      return isValidGenericDocument(number);
    case 'OTHER':
      return isValidGenericDocument(number, 1, 30);
    default:
      return false;
  }
}
