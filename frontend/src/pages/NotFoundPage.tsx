import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center">
      <h1 className="text-4xl font-bold text-gray-800">404</h1>
      <p className="mt-2 text-gray-500">Pagina nao encontrada</p>
      <Link to="/" className="mt-4 text-sm text-blue-600 hover:underline">
        Voltar ao Dashboard
      </Link>
    </div>
  );
}
