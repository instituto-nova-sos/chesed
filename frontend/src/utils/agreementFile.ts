export const AGREEMENT_ACCEPTED_TYPES = ['application/pdf', 'image/jpeg', 'image/png'];
export const AGREEMENT_MAX_SIZE_MB = 10;

/**
 * Validates a volunteer-agreement document against the accepted MIME types and size
 * limit shared by the coordinator upload modal and the self-service agreement page.
 * Returns a Portuguese error message, or null when the file is acceptable.
 */
export function validateAgreementFile(file: File): string | null {
  if (!AGREEMENT_ACCEPTED_TYPES.includes(file.type)) {
    return 'Formato invalido. Aceitos: PDF, JPEG, PNG.';
  }
  if (file.size > AGREEMENT_MAX_SIZE_MB * 1024 * 1024) {
    return `Arquivo excede o limite de ${AGREEMENT_MAX_SIZE_MB}MB.`;
  }
  return null;
}
