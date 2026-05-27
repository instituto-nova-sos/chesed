/**
 * Offline Behavior: Category C — Online-Only
 * - Requires network for agreement acceptance/rejection
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useState, useEffect } from 'react';
import { getAgreementText, acceptAgreement, rejectAgreement } from '../api/persons';
import { useAuth } from '../hooks/useAuth';
import { Button } from '../components/ui/Button';
import { Alert } from '../components/ui/Alert';
import { keycloak } from '../auth/keycloak';

export function VolunteerAgreementPage() {
  const { logout } = useAuth();
  const [agreementText, setAgreementText] = useState('');
  const [version, setVersion] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [rejected, setRejected] = useState(false);

  useEffect(() => {
    getAgreementText()
      .then(({ text, version: v }) => {
        setAgreementText(text);
        setVersion(v);
      })
      .catch(() => setError('Erro ao carregar o termo. Tente novamente.'))
      .finally(() => setLoading(false));
  }, []);

  const handleAccept = async () => {
    setSubmitting(true);
    setError(null);
    try {
      await acceptAgreement();
      // Force re-login so token claims are refreshed
      await keycloak.logout({ redirectUri: window.location.origin });
    } catch {
      setError('Erro ao aceitar o termo. Tente novamente.');
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    setSubmitting(true);
    setError(null);
    try {
      await rejectAgreement(rejectReason || undefined);
      setRejected(true);
    } catch {
      setError('Erro ao recusar o termo. Tente novamente.');
    } finally {
      setSubmitting(false);
    }
  };

  if (rejected) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
        <div className="max-w-md w-full bg-white rounded-lg shadow-md p-8 text-center">
          <div className="text-red-500 text-4xl mb-4">!</div>
          <h2 className="text-xl font-semibold text-gray-900 mb-2">
            Termo Recusado
          </h2>
          <p className="text-gray-600 mb-6">
            Sem a aceite do termo de voluntario, o acesso a plataforma fica restrito.
            Entre em contato com a coordenacao para mais informacoes.
          </p>
          <Button onClick={() => logout()} variant="secondary">
            Sair
          </Button>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8 px-4">
      <div className="max-w-2xl mx-auto">
        <div className="bg-white rounded-lg shadow-md overflow-hidden">
          {/* Header */}
          <div className="bg-blue-600 px-6 py-4">
            <h1 className="text-xl font-semibold text-white">
              Termo de Voluntario
            </h1>
            <p className="text-blue-100 text-sm mt-1">
              Versao {version} - Leia com atencao antes de aceitar
            </p>
          </div>

          {/* Agreement text */}
          <div className="px-6 py-4">
            <div className="prose prose-sm max-w-none max-h-96 overflow-y-auto border border-gray-200 rounded-md p-4 bg-gray-50">
              <div className="whitespace-pre-wrap text-sm text-gray-700">
                {agreementText}
              </div>
            </div>
          </div>

          {error && (
            <div className="px-6">
              <Alert type="error">{error}</Alert>
            </div>
          )}

          {/* Reject confirmation */}
          {showRejectConfirm && (
            <div className="px-6 py-4 bg-red-50 border-t border-red-100">
              <p className="text-sm text-red-700 font-medium mb-2">
                Tem certeza que deseja recusar o termo?
              </p>
              <p className="text-sm text-red-600 mb-3">
                Ao recusar, seu acesso a plataforma sera restrito.
              </p>
              <textarea
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Motivo da recusa (opcional)"
                className="w-full border border-red-300 rounded-md p-2 text-sm mb-3 focus:ring-red-500 focus:border-red-500"
                rows={3}
              />
              <div className="flex gap-3">
                <Button
                  onClick={handleReject}
                  variant="danger"
                  disabled={submitting}
                >
                  {submitting ? 'Enviando...' : 'Confirmar Recusa'}
                </Button>
                <Button
                  onClick={() => setShowRejectConfirm(false)}
                  variant="secondary"
                  disabled={submitting}
                >
                  Cancelar
                </Button>
              </div>
            </div>
          )}

          {/* Actions */}
          {!showRejectConfirm && (
            <div className="px-6 py-4 border-t border-gray-200 flex justify-between">
              <Button
                onClick={() => setShowRejectConfirm(true)}
                variant="secondary"
                disabled={submitting}
              >
                Recusar
              </Button>
              <Button
                onClick={handleAccept}
                disabled={submitting}
              >
                {submitting ? 'Processando...' : 'Aceitar Termo'}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
