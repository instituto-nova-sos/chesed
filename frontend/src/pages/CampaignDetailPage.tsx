import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useCampaign } from '../hooks/useCampaign';
import { useCampaignMetrics } from '../hooks/useCampaignMetrics';
import { useAuth } from '../hooks/useAuth';
import { usePersons } from '../hooks/usePersons';
import { ApiError } from '../api/client';
import { Badge } from '../components/ui/Badge';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { LoadingScreen } from '../components/ui/LoadingScreen';
import { SearchableSelect } from '../components/ui/SearchableSelect';
import { CampaignMetricsCards } from '../components/campaigns/CampaignMetricsCards';
import { CampaignTeamList } from '../components/campaigns/CampaignTeamList';
import {
  CAMPAIGN_ROLE_LABELS,
  campaignStatusLabel,
  campaignTypeLabel,
} from '../utils/campaignLabels';
import type { CampaignDetail } from '../types';

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function teamErrorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 409) {
    return 'Esta pessoa já faz parte da equipe da campanha.';
  }
  if (err instanceof ApiError && err.status === 400) {
    return 'Não foi possível adicionar: verifique a pessoa selecionada.';
  }
  return 'Erro ao atualizar a equipe. Tente novamente.';
}

function CampaignInfo({ campaign }: { campaign: CampaignDetail }) {
  const rows = [
    { label: 'Tipo', value: campaignTypeLabel(campaign.campaign_type) },
    { label: 'Início', value: formatDate(campaign.start_date) },
    { label: 'Término', value: campaign.end_date ? formatDate(campaign.end_date) : '—' },
    { label: 'Local', value: campaign.location_name ?? '—' },
    { label: 'Endereço', value: campaign.location_address ?? '—' },
    { label: 'Descrição', value: campaign.description ?? '—' },
  ];
  return (
    <dl className="grid grid-cols-1 gap-3 rounded-lg border border-gray-200 bg-white p-4 sm:grid-cols-2">
      {rows.map((row) => (
        <div key={row.label}>
          <dt className="text-xs font-medium uppercase text-gray-500">{row.label}</dt>
          <dd className="text-sm text-gray-900">{row.value}</dd>
        </div>
      ))}
    </dl>
  );
}

interface AddMemberFormProps {
  onAdd: (personId: string, role: string) => Promise<void>;
  submitting: boolean;
}

function AddMemberForm({ onAdd, submitting }: AddMemberFormProps) {
  const { persons } = usePersons();
  const [personId, setPersonId] = useState('');
  const [role, setRole] = useState('VOLUNTEER');

  return (
    <div className="space-y-3 rounded-lg border border-gray-200 bg-white p-4">
      <SearchableSelect
        label="Pessoa"
        options={persons.map((p) => ({ value: p.id, label: p.full_name }))}
        value={personId}
        onChange={setPersonId}
      />
      <div>
        <label htmlFor="team-role" className="block text-sm font-medium text-gray-700">
          Função na campanha
        </label>
        <select
          id="team-role"
          className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm"
          value={role}
          onChange={(e) => setRole(e.target.value)}
        >
          {Object.entries(CAMPAIGN_ROLE_LABELS).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>
      <Button
        type="button"
        disabled={!personId || submitting}
        onClick={() => void onAdd(personId, role).then(() => setPersonId(''))}
      >
        Adicionar
      </Button>
    </div>
  );
}

interface TeamMutations {
  addMember: (input: { person_id: string; role_in_campaign: string }) => Promise<void>;
  removeMember: (personId: string) => Promise<void>;
}

// Wraps the team mutations with the UI state transitions (error message,
// in-flight flags) so the page component stays under the complexity budget.
function useTeamActions({ addMember, removeMember }: TeamMutations) {
  const [teamError, setTeamError] = useState<string | null>(null);
  const [teamSubmitting, setTeamSubmitting] = useState(false);
  const [removingId, setRemovingId] = useState<string | null>(null);

  async function handleAdd(personId: string, role: string) {
    setTeamError(null);
    setTeamSubmitting(true);
    try {
      await addMember({ person_id: personId, role_in_campaign: role });
    } catch (err: unknown) {
      setTeamError(teamErrorMessage(err));
    } finally {
      setTeamSubmitting(false);
    }
  }

  async function handleRemove(personId: string) {
    setTeamError(null);
    setRemovingId(personId);
    try {
      await removeMember(personId);
    } catch (err: unknown) {
      setTeamError(teamErrorMessage(err));
    } finally {
      setRemovingId(null);
    }
  }

  return { teamError, teamSubmitting, removingId, handleAdd, handleRemove };
}

function MetricsSection({ metrics }: { metrics: ReturnType<typeof useCampaignMetrics> }) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-semibold text-gray-900">Indicadores</h2>
      {metrics.error && <Alert variant="info">{metrics.error}</Alert>}
      {metrics.metrics && <CampaignMetricsCards metrics={metrics.metrics} />}
    </section>
  );
}

interface TeamSectionProps {
  campaign: CampaignDetail;
  canManage: boolean;
  team: ReturnType<typeof useTeamActions>;
}

function TeamSection({ campaign, canManage, team }: TeamSectionProps) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-semibold text-gray-900">Equipe</h2>
      {team.teamError && <Alert variant="error">{team.teamError}</Alert>}
      <CampaignTeamList
        team={campaign.team}
        canManage={canManage}
        onRemove={(personId) => void team.handleRemove(personId)}
        removingId={team.removingId}
      />
      {canManage && <AddMemberForm onAdd={team.handleAdd} submitting={team.teamSubmitting} />}
    </section>
  );
}

export function CampaignDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { hasMinRole } = useAuth();
  const canManage = hasMinRole('COORDINATOR');
  const { campaign, isLoading, error, addMember, removeMember } = useCampaign(id);
  const metrics = useCampaignMetrics(id, canManage);
  const team = useTeamActions({ addMember, removeMember });

  if (isLoading) return <LoadingScreen />;
  if (error || !campaign) {
    return <Alert variant="error">{error ?? 'Campanha não encontrada'}</Alert>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold text-gray-900">{campaign.name}</h1>
          <Badge label={campaignStatusLabel(campaign.status)} variant={campaign.status} />
        </div>
        {canManage && (
          <Link
            to={`/campaigns/${campaign.id}/edit`}
            className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Editar
          </Link>
        )}
      </div>

      <CampaignInfo campaign={campaign} />
      {canManage && <MetricsSection metrics={metrics} />}
      <TeamSection campaign={campaign} canManage={canManage} team={team} />
    </div>
  );
}
