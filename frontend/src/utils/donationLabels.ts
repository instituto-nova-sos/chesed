export const DONATION_TYPE_LABELS: Record<string, string> = {
  FINANCIAL: 'Financeira',
  GOODS: 'Bens',
  SERVICES: 'Serviços',
};

export function donationTypeLabel(type: string): string {
  return DONATION_TYPE_LABELS[type] ?? type;
}

export function formatCurrency(amount: number | null, currency: string): string {
  if (amount === null) return '—';
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
  }).format(amount);
}
