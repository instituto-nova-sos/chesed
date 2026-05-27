import { z } from 'zod';
import { isValidCPF } from './cpfValidation';

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
  .refine(
    (data) => {
      if (
        data.document_type === 'CPF' &&
        data.document_number &&
        data.document_number.length > 0
      ) {
        return isValidCPF(data.document_number);
      }
      return true;
    },
    { message: 'CPF inválido', path: ['document_number'] },
  );

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
  .refine(
    (data) => {
      if (
        data.document_type === 'CPF' &&
        data.document_number &&
        data.document_number.length > 0
      ) {
        return isValidCPF(data.document_number);
      }
      return true;
    },
    { message: 'CPF inválido', path: ['document_number'] },
  );

export type CreatePersonFormData = z.infer<typeof createPersonSchema>;
export type UpdatePersonFormData = z.infer<typeof updatePersonSchema>;
export type SelfRegisterFormData = z.infer<typeof selfRegisterSchema>;
