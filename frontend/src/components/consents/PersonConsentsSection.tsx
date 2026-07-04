import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { useOfflineStatus } from '../../hooks/useOfflineStatus';
import { usePersonConsents } from '../../hooks/usePersonConsents';
import { Alert } from '../ui/Alert';
import { ConsentHistoryList } from './ConsentHistoryList';
import { RevokeConsentDialog } from './RevokeConsentDialog';
import type { Consent } from '../../types/consent';

interface PersonConsentsSectionProps {
  personId: string;
}

// Wraps the revoke mutation with dialog state so the section component stays
// under the complexity budget.
function useRevokeAction(revoke: (consentId: string, reason: string) => Promise<void>) {
  const [target, setTarget] = useState<Consent | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function confirm(reason: string) {
    if (!target) return;
    setSubmitting(true);
    setError(null);
    try {
      await revoke(target.id, reason);
      setTarget(null);
    } catch {
      setError('Erro ao revogar consentimento. Tente novamente.');
    } finally {
      setSubmitting(false);
    }
  }

  function cancel() {
    setTarget(null);
    setError(null);
  }

  return { target, setTarget, submitting, error, confirm, cancel };
}

export function PersonConsentsSection({ personId }: PersonConsentsSectionProps) {
  const { hasMinRole, hasRole } = useAuth();
  const { isOffline } = useOfflineStatus();
  const canView = hasMinRole('SECRETARY');
  const canRevoke = hasRole('ADMIN');
  // Consents are CRITICAL PII: never fetched (nor cached) while offline, and
  // never fetched for roles below SECRETARY.
  const enabled = canView && !isOffline;
  const { consents, isLoading, error, revoke } = usePersonConsents(
    enabled ? personId : undefined,
  );
  const revokeAction = useRevokeAction(revoke);

  if (!canView) return null;

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-gray-900">Consentimentos</h2>
        {!isOffline && (
          <Link
            to={`/persons/${personId}/consents/new`}
            className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Novo consentimento
          </Link>
        )}
      </div>

      {isOffline ? (
        <Alert variant="info">Conecte-se para ver os consentimentos.</Alert>
      ) : (
        <>
          {error && <Alert variant="error">{error}</Alert>}
          {isLoading ? (
            <p className="text-sm text-gray-500">Carregando…</p>
          ) : (
            <ConsentHistoryList
              consents={consents}
              canRevoke={canRevoke}
              onRevoke={revokeAction.setTarget}
            />
          )}
        </>
      )}

      {revokeAction.target && (
        <RevokeConsentDialog
          onCancel={revokeAction.cancel}
          onConfirm={(reason) => void revokeAction.confirm(reason)}
          submitting={revokeAction.submitting}
          error={revokeAction.error}
        />
      )}
    </section>
  );
}
