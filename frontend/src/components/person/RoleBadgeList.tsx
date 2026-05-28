import { useState } from 'react';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { useAuth } from '../../hooks/useAuth';
import { togglePersonRole } from '../../api/persons';
import { AddRoleModal } from './AddRoleModal';
import type { PersonRole } from '../../types';

interface RoleBadgeListProps {
  personId: string;
  roles: PersonRole[];
  onRolesChanged: () => void;
}

const roleLabels: Record<string, string> = {
  VOLUNTEER: 'Voluntário',
  ASSISTED: 'Assistido',
  PROFESSIONAL: 'Profissional',
  COORDINATOR: 'Coordenador',
  ADMIN: 'Administrador',
};

export function RoleBadgeList({
  personId,
  roles,
  onRolesChanged,
}: RoleBadgeListProps) {
  const { hasMinRole } = useAuth();
  const canManageRoles = hasMinRole('COORDINATOR');
  const [showAddModal, setShowAddModal] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  async function handleToggle(role: PersonRole) {
    setTogglingId(role.id);
    try {
      await togglePersonRole(personId, role.id, !role.is_active);
      onRolesChanged();
    } finally {
      setTogglingId(null);
    }
  }

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-900">Papéis</h3>
        {canManageRoles && (
          <Button size="sm" variant="secondary" onClick={() => setShowAddModal(true)}>
            Adicionar
          </Button>
        )}
      </div>

      {roles.length === 0 ? (
        <p className="mt-3 text-sm text-gray-500">Nenhum papel atribuído.</p>
      ) : (
        <div className="mt-3 space-y-2">
          {roles.map((role) => (
            <div
              key={role.id}
              className="flex items-center justify-between rounded-md border border-gray-100 px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <Badge
                  label={roleLabels[role.role_type] || role.role_type}
                  variant={role.role_type}
                />
                {!role.is_active && (
                  <span className="text-xs text-gray-400">(inativo)</span>
                )}
                {role.professional_specialty && (
                  <span className="text-xs text-gray-500">
                    {role.professional_specialty}
                  </span>
                )}
              </div>
              {canManageRoles && (
                <Button
                  size="sm"
                  variant={role.is_active ? 'danger' : 'secondary'}
                  isLoading={togglingId === role.id}
                  onClick={() => handleToggle(role)}
                >
                  {role.is_active ? 'Desativar' : 'Ativar'}
                </Button>
              )}
            </div>
          ))}
        </div>
      )}

      {showAddModal && (
        <AddRoleModal
          personId={personId}
          onClose={() => setShowAddModal(false)}
          onAdded={onRolesChanged}
        />
      )}
    </div>
  );
}
