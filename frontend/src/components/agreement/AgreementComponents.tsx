import { Button } from '../ui/Button';
import { SignaturePadCanvas } from '../ui/SignaturePadCanvas';

export type AcceptMethod = 'sign' | 'attach';

export function RejectedScreen({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow-md p-8 text-center">
        <div className="text-red-500 text-4xl mb-4">!</div>
        <h2 className="text-xl font-semibold text-gray-900 mb-2">Termo Recusado</h2>
        <p className="text-gray-600 mb-6">
          Sem a aceite do termo de voluntario, o acesso a plataforma fica restrito.
          Entre em contato com a coordenacao para mais informacoes.
        </p>
        <Button onClick={onLogout} variant="secondary">
          Sair
        </Button>
      </div>
    </div>
  );
}

interface RejectConfirmProps {
  rejectReason: string;
  onReasonChange: (value: string) => void;
  onConfirm: () => void;
  onCancel: () => void;
  submitting: boolean;
}

export function RejectConfirmBlock({
  rejectReason,
  onReasonChange,
  onConfirm,
  onCancel,
  submitting,
}: RejectConfirmProps) {
  return (
    <div className="px-6 py-4 bg-red-50 border-t border-red-100">
      <p className="text-sm text-red-700 font-medium mb-2">
        Tem certeza que deseja recusar o termo?
      </p>
      <p className="text-sm text-red-600 mb-3">
        Ao recusar, seu acesso a plataforma sera restrito.
      </p>
      <textarea
        value={rejectReason}
        onChange={(e) => onReasonChange(e.target.value)}
        placeholder="Motivo da recusa (opcional)"
        className="w-full border border-red-300 rounded-md p-2 text-sm mb-3 focus:ring-red-500 focus:border-red-500"
        rows={3}
      />
      <div className="flex gap-3">
        <Button onClick={onConfirm} variant="danger" disabled={submitting}>
          {submitting ? 'Enviando...' : 'Confirmar Recusa'}
        </Button>
        <Button onClick={onCancel} variant="secondary" disabled={submitting}>
          Cancelar
        </Button>
      </div>
    </div>
  );
}

interface AgreementActionsProps {
  onReject: () => void;
  onAccept: () => void;
  submitting: boolean;
  canSubmit: boolean;
}

export function AgreementActions({ onReject, onAccept, submitting, canSubmit }: AgreementActionsProps) {
  return (
    <div className="px-6 py-4 border-t border-gray-200 flex justify-between">
      <Button onClick={onReject} variant="secondary" disabled={submitting}>
        Recusar
      </Button>
      <Button onClick={onAccept} disabled={submitting || !canSubmit}>
        {submitting ? 'Processando...' : 'Aceitar Termo'}
      </Button>
    </div>
  );
}

interface AcceptMethodBlockProps {
  method: AcceptMethod;
  onSelectMethod: (m: AcceptMethod) => void;
  signature: string | null;
  onSign: (v: string | null) => void;
  onSelectFile: (f: File | null) => void;
  disabled: boolean;
}

export function AcceptMethodBlock({
  method,
  onSelectMethod,
  signature,
  onSign,
  onSelectFile,
  disabled,
}: AcceptMethodBlockProps) {
  return (
    <div className="px-6 py-4 border-t border-gray-100 space-y-4">
      <fieldset className="space-y-2">
        <legend className="text-sm font-medium text-gray-700">Como deseja aceitar?</legend>
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <input
            type="radio"
            name="accept-method"
            checked={method === 'sign'}
            onChange={() => onSelectMethod('sign')}
            disabled={disabled}
          />
          Assinar digitalmente
        </label>
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <input
            type="radio"
            name="accept-method"
            checked={method === 'attach'}
            onChange={() => onSelectMethod('attach')}
            disabled={disabled}
          />
          Anexar documento
        </label>
      </fieldset>

      {method === 'sign' ? (
        <SignaturePadCanvas value={signature ?? undefined} onChange={onSign} disabled={disabled} />
      ) : (
        <div>
          <label htmlFor="agreement-doc" className="block text-sm font-medium text-gray-700">
            Documento assinado (PDF, JPEG ou PNG)
          </label>
          <input
            id="agreement-doc"
            type="file"
            accept=".pdf,.jpg,.jpeg,.png"
            disabled={disabled}
            onChange={(e) => onSelectFile(e.target.files?.[0] ?? null)}
            className="mt-1 block w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
          />
        </div>
      )}
    </div>
  );
}
