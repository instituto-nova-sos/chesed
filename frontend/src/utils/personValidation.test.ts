import { describe, expect, it } from 'vitest';
import { createPersonSchema, selfRegisterSchema } from './personValidation';

function baseCreate(overrides: Record<string, unknown> = {}) {
  return {
    full_name: 'Maria da Silva',
    document_type: 'CPF',
    document_number: '',
    ...overrides,
  };
}

function baseSelfRegister(overrides: Record<string, unknown> = {}) {
  return {
    full_name: 'Maria da Silva',
    document_type: 'CPF',
    document_number: '',
    role_type: 'VOLUNTEER',
    campus_id: '11111111-1111-4111-8111-111111111111',
    ...overrides,
  };
}

function documentError(result: ReturnType<typeof createPersonSchema.safeParse>) {
  if (result.success) return undefined;
  return result.error.issues.find((i) => i.path[0] === 'document_number')?.message;
}

describe('createPersonSchema document validation', () => {
  it('accepts an empty document number for any type', () => {
    expect(createPersonSchema.safeParse(baseCreate({ document_type: 'PASSPORT' })).success).toBe(
      true,
    );
  });

  it('rejects an invalid CPF with the CPF message', () => {
    const result = createPersonSchema.safeParse(
      baseCreate({ document_type: 'CPF', document_number: '111.111.111-11' }),
    );
    expect(result.success).toBe(false);
    expect(documentError(result)).toBe('CPF inválido');
  });

  it('accepts a valid CPF', () => {
    expect(
      createPersonSchema.safeParse(
        baseCreate({ document_type: 'CPF', document_number: '529.982.247-25' }),
      ).success,
    ).toBe(true);
  });

  it('rejects a malformed SSN with the generic message', () => {
    const result = createPersonSchema.safeParse(
      baseCreate({ document_type: 'SSN', document_number: '12-345-6789' }),
    );
    expect(result.success).toBe(false);
    expect(documentError(result)).toBe('Documento inválido');
  });

  it('accepts a valid SSN', () => {
    expect(
      createPersonSchema.safeParse(
        baseCreate({ document_type: 'SSN', document_number: '123-45-6789' }),
      ).success,
    ).toBe(true);
  });

  it('rejects a too-short PASSPORT with the generic message', () => {
    const result = createPersonSchema.safeParse(
      baseCreate({ document_type: 'PASSPORT', document_number: 'ab' }),
    );
    expect(result.success).toBe(false);
    expect(documentError(result)).toBe('Documento inválido');
  });

  it('accepts a single-character OTHER document', () => {
    expect(
      createPersonSchema.safeParse(
        baseCreate({ document_type: 'OTHER', document_number: 'X' }),
      ).success,
    ).toBe(true);
  });
});

describe('selfRegisterSchema document validation', () => {
  it('rejects a malformed SSN with the generic message', () => {
    const result = selfRegisterSchema.safeParse(
      baseSelfRegister({ document_type: 'SSN', document_number: '12-345-6789' }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(
        result.error.issues.find((i) => i.path[0] === 'document_number')?.message,
      ).toBe('Documento inválido');
    }
  });

  it('accepts a valid EU_ID', () => {
    expect(
      selfRegisterSchema.safeParse(
        baseSelfRegister({ document_type: 'EU_ID', document_number: 'DE123456789' }),
      ).success,
    ).toBe(true);
  });
});
