import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

const getAgreementText = vi.fn();
const acceptAgreement = vi.fn();
const uploadAgreementSelf = vi.fn();
const rejectAgreement = vi.fn();
vi.mock('../../api/persons', () => ({
  getAgreementText: (...a: unknown[]) => getAgreementText(...a),
  acceptAgreement: (...a: unknown[]) => acceptAgreement(...a),
  uploadAgreementSelf: (...a: unknown[]) => uploadAgreementSelf(...a),
  rejectAgreement: (...a: unknown[]) => rejectAgreement(...a),
}));

const logout = vi.fn();
vi.mock('../../auth/keycloak', () => ({
  keycloak: { logout: (...a: unknown[]) => logout(...a) },
}));

vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({ logout: vi.fn() }),
}));

// Stub the canvas so "signing" is a deterministic click in jsdom.
vi.mock('../../components/ui/SignaturePadCanvas', () => ({
  SignaturePadCanvas: (props: { onChange: (v: string | null) => void; disabled?: boolean }) => (
    <button
      type="button"
      disabled={props.disabled}
      onClick={() => props.onChange('data:image/png;base64,STUB')}
    >
      stub-sign
    </button>
  ),
}));

import { VolunteerAgreementPage } from '../VolunteerAgreementPage';

async function renderLoaded() {
  getAgreementText.mockResolvedValue({ text: 'Termo de teste', version: 'v1' });
  render(<VolunteerAgreementPage />);
  await screen.findByText('Termo de teste');
}

describe('VolunteerAgreementPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('blocks digital acceptance until a signature is drawn', async () => {
    await renderLoaded();
    const acceptBtn = screen.getByRole('button', { name: /aceitar termo/i });
    expect(acceptBtn).toBeDisabled();
    expect(acceptAgreement).not.toHaveBeenCalled();
  });

  it('sends the drawn signature on digital acceptance', async () => {
    acceptAgreement.mockResolvedValue({ id: 'a1', status: 'ACCEPTED' });
    await renderLoaded();

    await userEvent.click(screen.getByRole('button', { name: /stub-sign/i }));
    const acceptBtn = screen.getByRole('button', { name: /aceitar termo/i });
    await waitFor(() => expect(acceptBtn).toBeEnabled());
    await userEvent.click(acceptBtn);

    expect(acceptAgreement).toHaveBeenCalledWith('data:image/png;base64,STUB');
  });

  it('lets the volunteer attach a document instead and blocks until a file is chosen', async () => {
    uploadAgreementSelf.mockResolvedValue({ id: 'a1', status: 'ACCEPTED' });
    await renderLoaded();

    // Switch to the attach method.
    await userEvent.click(screen.getByRole('radio', { name: /anexar documento/i }));

    const acceptBtn = screen.getByRole('button', { name: /aceitar termo/i });
    expect(acceptBtn).toBeDisabled();

    const file = new File(['%PDF-1.4'], 'termo.pdf', { type: 'application/pdf' });
    const input = screen.getByLabelText(/documento assinado/i);
    await userEvent.upload(input, file);

    await waitFor(() => expect(acceptBtn).toBeEnabled());
    await userEvent.click(acceptBtn);

    expect(uploadAgreementSelf).toHaveBeenCalledWith(file);
    expect(acceptAgreement).not.toHaveBeenCalled();
  });

  it('rejects an invalid attachment MIME type', async () => {
    await renderLoaded();
    await userEvent.click(screen.getByRole('radio', { name: /anexar documento/i }));

    const bad = new File(['x'], 'termo.txt', { type: 'text/plain' });
    const input = screen.getByLabelText(/documento assinado/i);
    // applyAccept:false bypasses the input's `accept` filter so the component's own
    // MIME validation (the real defense) is exercised.
    await userEvent.upload(input, bad, { applyAccept: false });

    expect(await screen.findByText(/formato invalido/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /aceitar termo/i })).toBeDisabled();
  });
});
