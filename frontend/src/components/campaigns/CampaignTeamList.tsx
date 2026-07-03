import type { CampaignTeamMember } from '../../types';
import { campaignRoleLabel } from '../../utils/campaignLabels';
import { Badge } from '../ui/Badge';

interface CampaignTeamListProps {
  team: CampaignTeamMember[];
  canManage: boolean;
  onRemove: (personId: string) => void;
  removingId: string | null;
}

export function CampaignTeamList({ team, canManage, onRemove, removingId }: CampaignTeamListProps) {
  if (team.length === 0) {
    return <p className="text-sm text-gray-500">Nenhum membro na equipe ainda.</p>;
  }

  return (
    <ul className="divide-y divide-gray-100 rounded-lg border border-gray-200 bg-white">
      {team.map((member) => (
        <li key={member.person_id} className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-gray-900">{member.person_name}</span>
            <Badge
              label={campaignRoleLabel(member.role_in_campaign)}
              variant={member.role_in_campaign}
            />
          </div>
          {canManage && (
            <button
              type="button"
              onClick={() => onRemove(member.person_id)}
              disabled={removingId === member.person_id}
              className="text-sm font-medium text-red-600 hover:text-red-800 disabled:opacity-50"
            >
              Remover
            </button>
          )}
        </li>
      ))}
    </ul>
  );
}
