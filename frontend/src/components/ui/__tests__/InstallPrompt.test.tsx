import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { InstallPrompt } from '../InstallPrompt';

interface BeforeInstallPromptEventLike extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

function fireBeforeInstallPrompt(): { prompt: ReturnType<typeof vi.fn> } {
  const prompt = vi.fn().mockResolvedValue(undefined);
  const event = new Event('beforeinstallprompt') as BeforeInstallPromptEventLike;
  event.preventDefault = vi.fn();
  event.prompt = prompt;
  event.userChoice = Promise.resolve({ outcome: 'accepted' });
  act(() => {
    window.dispatchEvent(event);
  });
  return { prompt };
}

describe('InstallPrompt', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders nothing until the browser offers installation', () => {
    const { container } = render(<InstallPrompt />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows an install button once beforeinstallprompt fires', async () => {
    render(<InstallPrompt />);
    fireBeforeInstallPrompt();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /instalar/i })).toBeInTheDocument(),
    );
  });

  it('calls the deferred prompt and hides itself when the button is clicked', async () => {
    render(<InstallPrompt />);
    const { prompt } = fireBeforeInstallPrompt();
    const button = await screen.findByRole('button', { name: /instalar/i });
    await userEvent.click(button);
    expect(prompt).toHaveBeenCalledOnce();
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /instalar/i })).not.toBeInTheDocument(),
    );
  });
});
