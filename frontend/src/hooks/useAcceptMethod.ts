import { useState } from 'react';
import { validateAgreementFile } from '../utils/agreementFile';
import type { AcceptMethod } from '../components/agreement/AgreementComponents';

/**
 * Manages the volunteer-agreement acceptance method state: the chosen method
 * (draw a signature vs. attach a document), the drawn signature, and the selected
 * file. Switching method clears the other input so the two cannot diverge, and file
 * selection runs the shared MIME/size validation. Returns a validation error string
 * to surface, or null.
 */
export function useAcceptMethod() {
  const [method, setMethod] = useState<AcceptMethod>('sign');
  const [signature, setSignature] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);

  const selectMethod = (next: AcceptMethod) => {
    setMethod(next);
    setSignature(null);
    setFile(null);
    setFileError(null);
  };

  const selectFile = (selected: File | null) => {
    setFileError(null);
    if (!selected) {
      setFile(null);
      return;
    }
    const validationError = validateAgreementFile(selected);
    if (validationError) {
      setFileError(validationError);
      setFile(null);
      return;
    }
    setFile(selected);
  };

  const canSubmit = method === 'sign' ? signature !== null : file !== null;

  return {
    method,
    selectMethod,
    signature,
    setSignature,
    file,
    selectFile,
    fileError,
    canSubmit,
  };
}
