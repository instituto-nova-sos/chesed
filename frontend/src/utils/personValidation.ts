import { z } from 'zod';
import { isValidDocumentFormat } from './documentFormat';

export const addressSchema = z.object({
  zip_code: z.string().max(20).optional().or(z.literal('')),
  street: z.string().max(300).optional().or(z.literal('')),
  number: z.string().max(20).optional().or(z.literal('')),
  complement: z.string().max(100).optional().or(z.literal('')),
  neighborhood: z.string().max(100).optional().or(z.literal('')),
  city: z.string().max(100).optional().or(z.literal('')),
  state: z.string().max(100).optional().or(z.literal('')),
  country: z.string().max(3).optional().or(z.literal('')),
});

interface DocumentBearing {
  document_type: string;
  document_number?: string;
}

function validateDocumentNumber(data: DocumentBearing, ctx: z.RefinementCtx): void {
  const number = data.document_number ?? '';
  if (number.length === 0) return;
  if (isValidDocumentFormat(data.document_type, number)) return;
  ctx.addIssue({
    code: z.ZodIssueCode.custom,
    path: ['document_number'],
    message: data.document_type === 'CPF' ? 'CPF inválido' : 'Documento inválido',
  });
}

export const createPersonSchema = z
  .object({
    full_name: z.string().min(1, 'Nome completo é obrigatório').max(200),
    birth_date: z.string().optional().or(z.literal('')),
    document_type: z.enum(['CPF', 'RG', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER']),
    document_number: z.string().max(30).optional().or(z.literal('')),
    nationality: z.string().length(3).optional().or(z.literal('')),
    gender: z
      .enum(['M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY'])
      .optional()
      .or(z.literal('')),
    email: z
      .string()
      .email('Email inválido')
      .max(255)
      .optional()
      .or(z.literal('')),
    phone: z.string().max(30).optional().or(z.literal('')),
    referral_source: z.string().max(200).optional().or(z.literal('')),
    address: addressSchema.optional(),
  })
  .superRefine(validateDocumentNumber);

export const updatePersonSchema = createPersonSchema;

export const selfRegisterSchema = z
  .object({
    full_name: z.string().min(1, 'Nome completo é obrigatório').max(200),
    birth_date: z.string().optional().or(z.literal('')),
    document_type: z.enum(['CPF', 'RG', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER']),
    document_number: z.string().max(30).optional().or(z.literal('')),
    nationality: z.string().length(3).optional().or(z.literal('')),
    gender: z
      .enum(['M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY'])
      .optional()
      .or(z.literal('')),
    phone: z.string().max(30).optional().or(z.literal('')),
    referral_source: z.string().max(200).optional().or(z.literal('')),
    address: addressSchema.optional(),
    role_type: z.enum(['VOLUNTEER', 'ASSISTED']),
    campus_id: z.string().uuid('Campus é obrigatório'),
  })
  .superRefine(validateDocumentNumber);

export type CreatePersonFormData = z.infer<typeof createPersonSchema>;
export type UpdatePersonFormData = z.infer<typeof updatePersonSchema>;
export type SelfRegisterFormData = z.infer<typeof selfRegisterSchema>;
