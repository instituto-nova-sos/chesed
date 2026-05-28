import { useOfflineStatus } from '../../hooks/useOfflineStatus';

export function OfflineBanner() {
  const { isOffline } = useOfflineStatus();

  if (!isOffline) return null;

  return (
    <div className="bg-yellow-50 border-b border-yellow-200 px-4 py-2 text-center text-sm text-yellow-800">
      Você está offline. As alterações serão sincronizadas quando a conexão
      retornar.
    </div>
  );
}
