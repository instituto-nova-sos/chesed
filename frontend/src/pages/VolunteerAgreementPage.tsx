/**
 * Offline Behavior: Category C — Online-Only
 * - Requires network for agreement acceptance/rejection
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useState, useEffect } from 'react';
import {
  getAgreementText,
  acceptAgreement,
  uploadAgreementSelf,
  rejectAgreement,
} from '../api/persons';
import { useAuth } from '../hooks/useAuth';
import { useAcceptMethod } from '../hooks/useAcceptMethod';
import { Alert } from '../components/ui/Alert';
import {
  RejectedScreen,
  RejectConfirmBlock,
  AgreementActions,
  AcceptMethodBlock,
} from '../components/agreement/AgreementComponents';
import { keycloak } from '../auth/keycloak';

function useVolunteerAgreement() {
  const [agreementText, setAgreementText] = useState('');
  const [version, setVersion] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [rejected, setRejected] = useState(false);
  const accept = useAcceptMethod();

  useEffect(() => {
    getAgreementText()
      .then(({ text, version: v }) => {
        setAgreementText(text);
        setVersion(v);
      })
      .catch(() => setSubmitError('Erro ao carregar o termo. Tente novamente.'))
      .finally(() => setLoading(false));
  }, []);

  const handleAccept = async () => {
    if (!accept.canSubmit) return;
    setSubmitting(true);
    setSubmitError(null);
    try {
      if (accept.method === 'sign') {
        await acceptAgreement(accept.signature as string);
      } else {
        await uploadAgreementSelf(accept.file as File);
      }
      // Force re-login so token claims are refreshed
      await keycloak.logout({ redirectUri: window.location.origin });
    } catch {
      setSubmitError('Erro ao aceitar o termo. Tente novamente.');
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    setSubmitting(true);
    setSubmitError(null);
    try {
      await rejectAgreement(rejectReason || undefined);
      setRejected(true);
    } catch {
      setSubmitError('Erro ao recusar o termo. Tente novamente.');
    } finally {
      setSubmitting(false);
    }
  };

  return {
    agreementText,
    version,
    loading,
    submitting,
    error: submitError ?? accept.fileError,
    rejectReason,
    setRejectReason,
    rejected,
    accept,
    handleAccept,
    handleReject,
  };
}

type AgreementState = ReturnType<typeof useVolunteerAgreement>;

function AgreementFooter({ state }: { state: AgreementState }) {
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);
  const { submitting, rejectReason, setRejectReason, accept, handleAccept, handleReject } = state;

  if (showRejectConfirm) {
    return (
      <RejectConfirmBlock
        rejectReason={rejectReason}
        onReasonChange={setRejectReason}
        onConfirm={handleReject}
        onCancel={() => setShowRejectConfirm(false)}
        submitting={submitting}
      />
    );
  }
  return (
    <>
      <AcceptMethodBlock
        method={accept.method}
        onSelectMethod={accept.selectMethod}
        signature={accept.signature}
        onSign={accept.setSignature}
        onSelectFile={accept.selectFile}
        disabled={submitting}
      />
      <AgreementActions
        onReject={() => setShowRejectConfirm(true)}
        onAccept={handleAccept}
        submitting={submitting}
        canSubmit={accept.canSubmit}
      />
    </>
  );
}

export function VolunteerAgreementPage() {
  const { logout } = useAuth();
  const state = useVolunteerAgreement();

  if (state.rejected) {
    return <RejectedScreen onLogout={() => logout()} />;
  }

  if (state.loading) {
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
          <div className="bg-blue-600 px-6 py-4">
            <h1 className="text-xl font-semibold text-white">Termo de Voluntario</h1>
            <p className="text-blue-100 text-sm mt-1">
              Versao {state.version} - Leia com atencao antes de aceitar
            </p>
          </div>

          <div className="px-6 py-4">
            <div className="prose prose-sm max-w-none max-h-96 overflow-y-auto border border-gray-200 rounded-md p-4 bg-gray-50">
              <div className="whitespace-pre-wrap text-sm text-gray-700">
                {state.agreementText}
              </div>
            </div>
          </div>

          {state.error && (
            <div className="px-6">
              <Alert variant="error">{state.error}</Alert>
            </div>
          )}

          <AgreementFooter state={state} />
        </div>
      </div>
    </div>
  );
}
