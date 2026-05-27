import { useAuth } from '../hooks/useAuth';
import { keycloak } from '../auth/keycloak';

export function EmailVerificationPendingPage() {
  const { email, logout } = useAuth();

  function handleRetry() {
    keycloak.login();
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 p-4">
      <div className="w-full max-w-md rounded-lg border border-gray-200 bg-white p-6 shadow-lg text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-amber-100">
          <svg
            className="h-6 w-6 text-amber-600"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75"
            />
          </svg>
        </div>

        <h1 className="text-xl font-semibold text-gray-900">
          Verificação de E-mail Pendente
        </h1>

        <p className="mt-3 text-sm text-gray-600">
          Para acessar o sistema, você precisa confirmar seu endereço de e-mail.
        </p>

        {email && (
          <p className="mt-2 text-sm font-medium text-gray-800">
            {email}
          </p>
        )}

        <div className="mt-4 rounded-lg bg-gray-50 p-4 text-left">
          <p className="text-sm font-medium text-gray-700 mb-2">
            Como confirmar:
          </p>
          <ol className="text-sm text-gray-600 space-y-1 list-decimal list-inside">
            <li>Verifique sua caixa de entrada</li>
            <li>Procure o e-mail de verificação</li>
            <li>Clique no link de confirmação</li>
            <li>Volte aqui e clique em "Tentar novamente"</li>
          </ol>
          <p className="mt-2 text-xs text-gray-500">
            Não encontrou? Verifique a pasta de spam ou lixo eletrônico.
          </p>
        </div>

        <div className="mt-6 flex flex-col gap-3">
          <button
            type="button"
            onClick={handleRetry}
            className="w-full rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Tentar novamente
          </button>
          <button
            type="button"
            onClick={logout}
            className="text-sm text-gray-500 hover:text-gray-700"
          >
            Sair
          </button>
        </div>
      </div>
    </div>
  );
}
