import type { ConsentType } from '../types/consent';

export const CONSENT_TYPE_LABELS: Record<ConsentType, string> = {
  DATA_PROCESSING: 'Tratamento de dados',
  IMAGE_USAGE: 'Uso de imagem',
  HEALTH_DATA: 'Dados de saúde',
  MINOR_GUARDIAN: 'Responsável por menor',
};

// LGPD Art. 8 requires the purpose to be shown at collection time; these
// presets prefill the form per type and remain editable by the operator.
export const PURPOSE_PRESETS: Record<ConsentType, string> = {
  DATA_PROCESSING:
    'Cadastro e acompanhamento de atendimentos e serviços prestados pelo Instituto Nova SOS.',
  IMAGE_USAGE:
    'Uso de imagem em materiais de divulgação e prestação de contas do Instituto Nova SOS.',
  HEALTH_DATA:
    'Registro de informações de saúde necessárias ao acompanhamento profissional.',
  MINOR_GUARDIAN:
    'Autorização do responsável legal para cadastro e atendimento de menor de idade.',
};

export const CONSENT_STATUS_LABELS = {
  active: 'Ativo',
  revoked: 'Revogado',
} as const;

export function consentTypeLabel(type: string): string {
  return (CONSENT_TYPE_LABELS as Record<string, string>)[type] ?? type;
}

export function consentStatusLabel(isActive: boolean): string {
  return isActive ? CONSENT_STATUS_LABELS.active : CONSENT_STATUS_LABELS.revoked;
}
