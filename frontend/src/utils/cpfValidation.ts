/**
 * Validates a Brazilian CPF number using the check-digit algorithm.
 * Accepts formatted (123.456.789-00) or unformatted (12345678900) input.
 */
export function isValidCPF(cpf: string): boolean {
  const digits = cpf.replace(/\D/g, '');

  if (digits.length !== 11) return false;

  // Reject known invalid sequences (all same digit)
  if (/^(\d)\1{10}$/.test(digits)) return false;

  const d = digits.split('').map(Number);

  // Calculate first check digit
  let sum = 0;
  for (let i = 0; i < 9; i++) {
    sum += d[i]! * (10 - i);
  }
  let remainder = sum % 11;
  const d1 = remainder < 2 ? 0 : 11 - remainder;

  if (d[9]! !== d1) return false;

  // Calculate second check digit
  sum = 0;
  for (let i = 0; i < 10; i++) {
    sum += d[i]! * (11 - i);
  }
  remainder = sum % 11;
  const d2 = remainder < 2 ? 0 : 11 - remainder;

  return d[10]! === d2;
}

/**
 * Formats a CPF string to the standard format: 123.456.789-00
 */
export function formatCPF(cpf: string): string {
  const digits = cpf.replace(/\D/g, '').slice(0, 11);
  if (digits.length <= 3) return digits;
  if (digits.length <= 6) return `${digits.slice(0, 3)}.${digits.slice(3)}`;
  if (digits.length <= 9)
    return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6)}`;
  return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6, 9)}-${digits.slice(9)}`;
}
