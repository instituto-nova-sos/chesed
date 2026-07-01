import { useEffect, useState } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

/**
 * InstallPrompt offers an "add to home screen" action once the browser fires
 * `beforeinstallprompt` (Chromium PWAs). It captures the deferred event, shows
 * a dismissible bar, and triggers the native prompt on demand. It renders
 * nothing on browsers that never fire the event (e.g. iOS Safari) so it degrades
 * gracefully.
 */
export function InstallPrompt() {
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);

  useEffect(() => {
    function onBeforeInstall(e: Event) {
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
    }
    function onInstalled() {
      setDeferred(null);
    }
    window.addEventListener('beforeinstallprompt', onBeforeInstall);
    window.addEventListener('appinstalled', onInstalled);
    return () => {
      window.removeEventListener('beforeinstallprompt', onBeforeInstall);
      window.removeEventListener('appinstalled', onInstalled);
    };
  }, []);

  if (!deferred) return null;

  async function install() {
    if (!deferred) return;
    await deferred.prompt();
    setDeferred(null);
  }

  return (
    <div className="flex items-center justify-between gap-3 border-b border-blue-200 bg-blue-600 px-4 py-2 text-sm text-white">
      <span>Instale o Chesed para acesso rápido e uso offline.</span>
      <div className="flex shrink-0 gap-2">
        <button
          type="button"
          onClick={() => void install()}
          className="rounded-md bg-white px-3 py-1 text-xs font-semibold text-blue-700 hover:bg-blue-50"
        >
          Instalar
        </button>
        <button
          type="button"
          onClick={() => setDeferred(null)}
          className="rounded-md px-2 py-1 text-xs font-medium text-blue-100 hover:text-white"
          aria-label="Dispensar"
        >
          Agora não
        </button>
      </div>
    </div>
  );
}
