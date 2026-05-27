/**
 * Offline Behavior: Category A — Fully Offline-Capable
 * - Person creation saves to IndexedDB + sync queue when offline
 * - Reference: docs/12-offline-sync-strategy.md
 */
import { useNavigate } from 'react-router-dom';
import { Button } from '../components/ui/Button';
import { PersonForm } from '../components/person/PersonForm';

export function PersonCreatePage() {
  const navigate = useNavigate();

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="secondary" size="sm" onClick={() => navigate('/persons')}>
          Voltar
        </Button>
        <h1 className="text-lg font-semibold text-gray-900">Nova Pessoa</h1>
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:p-6">
        <PersonForm />
      </div>
    </div>
  );
}
