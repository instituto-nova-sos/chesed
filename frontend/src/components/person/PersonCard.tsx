import { useNavigate } from 'react-router-dom';
import { Badge } from '../ui/Badge';
import type { PersonListItem } from '../../types';
import { formatPhoneDisplay } from '../../utils/phoneFormat';

interface PersonCardProps {
  person: PersonListItem;
}

const roleLabels: Record<string, string> = {
  VOLUNTEER: 'Voluntário',
  ASSISTED: 'Assistido',
  PROFESSIONAL: 'Profissional',
  COORDINATOR: 'Coordenador',
  ADMIN: 'Administrador',
};

export function PersonCard({ person }: PersonCardProps) {
  const navigate = useNavigate();

  return (
    <button
      type="button"
      onClick={() => navigate(`/persons/${person.id}`)}
      className="w-full rounded-lg border border-gray-200 bg-white p-4 text-left shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-medium text-gray-900">
            {person.full_name}
          </h3>
          {person.document_number && (
            <p className="mt-0.5 text-xs text-gray-500">
              {person.document_number}
            </p>
          )}
          {person.phone && (
            <p className="mt-0.5 text-xs text-gray-500">{formatPhoneDisplay(person.phone)}</p>
          )}
        </div>
      </div>
      {person.roles.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {person.roles.map((role) => (
            <Badge
              key={role}
              label={roleLabels[role] || role}
              variant={role}
            />
          ))}
        </div>
      )}
    </button>
  );
}
