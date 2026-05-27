import { useNavigate } from 'react-router-dom';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { useAuth } from '../../hooks/useAuth';
import type { PersonDetail } from '../../types';
import { formatPhoneDisplay } from '../../utils/phoneFormat';

interface PersonInfoProps {
  person: PersonDetail;
}

const roleLabels: Record<string, string> = {
  VOLUNTEER: 'Voluntário',
  ASSISTED: 'Assistido',
  PROFESSIONAL: 'Profissional',
  COORDINATOR: 'Coordenador',
  ADMIN: 'Administrador',
};

const genderLabels: Record<string, string> = {
  M: 'Masculino',
  F: 'Feminino',
  OTHER: 'Outro',
  PREFER_NOT_TO_SAY: 'Prefiro não dizer',
};

function formatDateBR(iso: string): string {
  const dateOnly = iso.split('T')[0];
  const [y, m, d] = dateOnly.split('-');
  return `${d}/${m}/${y}`;
}

export function PersonInfo({ person }: PersonInfoProps) {
  const navigate = useNavigate();
  const { hasMinRole } = useAuth();
  const canEdit = hasMinRole('SECRETARY');

  const address = person.addresses?.[0];

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900">
            {person.full_name}
          </h2>
          {person.document_number && (
            <p className="text-sm text-gray-500">
              {person.document_type}: {person.document_number}
            </p>
          )}
        </div>
        {canEdit && (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate(`/persons/${person.id}/edit`)}
          >
            Editar
          </Button>
        )}
      </div>

      {person.roles.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1">
          {person.roles
            .filter((r) => r.is_active)
            .map((role) => (
              <Badge
                key={role.id}
                label={roleLabels[role.role_type] || role.role_type}
                variant={role.role_type}
              />
            ))}
        </div>
      )}

      <div className="mt-4 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
        {person.birth_date && (
          <InfoField
            label="Data de Nascimento"
            value={formatDateBR(person.birth_date)}
          />
        )}
        {person.gender && (
          <InfoField
            label="Gênero"
            value={genderLabels[person.gender] || person.gender}
          />
        )}
        {person.email && <InfoField label="Email" value={person.email} />}
        {person.phone && <InfoField label="Telefone" value={formatPhoneDisplay(person.phone)} />}
        {person.referral_source && (
          <InfoField label="Fonte de Encaminhamento" value={person.referral_source} />
        )}
      </div>

      {address && (
        <div className="mt-4 border-t pt-3">
          <h3 className="text-sm font-medium text-gray-900">Endereço</h3>
          <p className="mt-1 text-sm text-gray-600">
            {[address.street, address.number].filter(Boolean).join(', ')}
            {address.complement && ` - ${address.complement}`}
          </p>
          <p className="text-sm text-gray-600">
            {[address.neighborhood, address.city, address.state]
              .filter(Boolean)
              .join(' - ')}
            {address.zip_code && ` | CEP: ${address.zip_code}`}
          </p>
        </div>
      )}
    </div>
  );
}

function InfoField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium text-gray-500">{label}</dt>
      <dd className="text-gray-900">{value}</dd>
    </div>
  );
}
