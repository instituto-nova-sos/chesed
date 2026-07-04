import { useState } from 'react';
import { Button } from '../ui/Button';

interface RevokeConsentDialogProps {
  onCancel: () => void;
  onConfirm: (reason: string) => void;
  submitting: boolean;
  error: string | null;
}

export function RevokeConsentDialog({
  onCancel,
  onConfirm,
  submitting,
  error,
}: RevokeConsentDialogProps) {
  const [reason, setReason] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);

  function handleConfirm() {
    const trimmed = reason.trim();
    if (!trimmed) {
      setValidationError('Informe o motivo da revogação.');
      return;
    }
    setValidationError(null);
    onConfirm(trimmed);
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="revoke-consent-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    >
      <div className="w-full max-w-sm space-y-3 rounded-lg bg-white p-4 shadow-lg">
        <h3 id="revoke-consent-title" className="text-base font-semibold text-gray-900">
          Revogar consentimento
        </h3>
        <div>
          <label
            htmlFor="revoke-reason"
            className="block text-sm font-medium text-gray-700"
          >
            Motivo da revogação
          </label>
          <textarea
            id="revoke-reason"
            rows={3}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {validationError && <p className="mt-1 text-xs text-red-600">{validationError}</p>}
        </div>
        {error && <p className="text-xs text-red-600">{error}</p>}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onCancel} disabled={submitting}>
            Cancelar
          </Button>
          <Button type="button" variant="danger" onClick={handleConfirm} disabled={submitting}>
            Confirmar revogação
          </Button>
        </div>
      </div>
    </div>
  );
}
