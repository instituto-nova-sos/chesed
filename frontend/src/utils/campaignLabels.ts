export const CAMPAIGN_TYPE_LABELS: Record<string, string> = {
  SOCIAL_ACTION: 'Ação Social',
  EDUCATIONAL: 'Educacional',
  HEALTH: 'Saúde',
  COMMUNITY: 'Comunitária',
  OTHER: 'Outra',
};

export const CAMPAIGN_STATUS_LABELS: Record<string, string> = {
  PLANNED: 'Planejada',
  ACTIVE: 'Ativa',
  COMPLETED: 'Concluída',
  CANCELLED: 'Cancelada',
};

export const CAMPAIGN_ROLE_LABELS: Record<string, string> = {
  COORDINATOR: 'Coordenação',
  PROFESSIONAL: 'Profissional',
  VOLUNTEER: 'Voluntário(a)',
  SUPPORT: 'Apoio',
};

export function campaignTypeLabel(type: string): string {
  return CAMPAIGN_TYPE_LABELS[type] ?? type;
}

export function campaignStatusLabel(status: string): string {
  return CAMPAIGN_STATUS_LABELS[status] ?? status;
}

export function campaignRoleLabel(role: string): string {
  return CAMPAIGN_ROLE_LABELS[role] ?? role;
}
