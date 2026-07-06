import { describe, expect, it } from 'vitest';
import {
  isValidSSN,
  isValidGenericDocument,
  isValidDocumentFormat,
  documentNumberPlaceholder,
} from './documentFormat';

describe('isValidSSN', () => {
  it('accepts a formatted SSN', () => {
    expect(isValidSSN('123-45-6789')).toBe(true);
  });

  it('accepts an unformatted SSN', () => {
    expect(isValidSSN('123456789')).toBe(true);
  });

  it('rejects too few digits', () => {
    expect(isValidSSN('123-45-678')).toBe(false);
  });

  it('rejects letters', () => {
    expect(isValidSSN('12A-45-6789')).toBe(false);
  });
});

describe('isValidGenericDocument', () => {
  it('accepts an alphanumeric value within length bounds', () => {
    expect(isValidGenericDocument('AB-123.456/78')).toBe(true);
  });

  it('rejects a value shorter than 3 characters', () => {
    expect(isValidGenericDocument('ab')).toBe(false);
  });

  it('rejects a value longer than 30 characters', () => {
    expect(isValidGenericDocument('A'.repeat(31))).toBe(false);
  });

  it('rejects disallowed symbols', () => {
    expect(isValidGenericDocument('ABC@123')).toBe(false);
  });
});

describe('isValidDocumentFormat', () => {
  it('returns true for an empty number regardless of type', () => {
    expect(isValidDocumentFormat('CPF', '')).toBe(true);
    expect(isValidDocumentFormat('PASSPORT', '')).toBe(true);
  });

  it('validates CPF via the CPF algorithm', () => {
    expect(isValidDocumentFormat('CPF', '529.982.247-25')).toBe(true);
    expect(isValidDocumentFormat('CPF', '111.111.111-11')).toBe(false);
  });

  it('validates SSN', () => {
    expect(isValidDocumentFormat('SSN', '123-45-6789')).toBe(true);
    expect(isValidDocumentFormat('SSN', '12-345-6789')).toBe(false);
  });

  it('validates RG generically', () => {
    expect(isValidDocumentFormat('RG', 'SSP/BA 1234567')).toBe(true);
    expect(isValidDocumentFormat('RG', 'x')).toBe(false);
  });

  it('validates EU_ID generically', () => {
    expect(isValidDocumentFormat('EU_ID', 'DE123456789')).toBe(true);
    expect(isValidDocumentFormat('EU_ID', 'no')).toBe(false);
  });

  it('validates PASSPORT generically', () => {
    expect(isValidDocumentFormat('PASSPORT', 'AB1234567')).toBe(true);
    expect(isValidDocumentFormat('PASSPORT', 'ab')).toBe(false);
  });

  it('accepts a single-character OTHER value', () => {
    expect(isValidDocumentFormat('OTHER', 'X')).toBe(true);
    expect(isValidDocumentFormat('OTHER', 'A'.repeat(31))).toBe(false);
  });

  it('rejects an unknown document type with a value', () => {
    expect(isValidDocumentFormat('UNKNOWN', 'anything')).toBe(false);
  });
});

describe('documentNumberPlaceholder', () => {
  it('returns the CPF mask', () => {
    expect(documentNumberPlaceholder('CPF')).toBe('000.000.000-00');
  });

  it('returns the SSN mask', () => {
    expect(documentNumberPlaceholder('SSN')).toBe('000-00-0000');
  });

  it('returns a passport example', () => {
    expect(documentNumberPlaceholder('PASSPORT')).toBe('AB123456');
  });

  it('returns a generic hint for other types', () => {
    expect(documentNumberPlaceholder('EU_ID')).toBe('Número do documento');
    expect(documentNumberPlaceholder('OTHER')).toBe('Número do documento');
  });
});
