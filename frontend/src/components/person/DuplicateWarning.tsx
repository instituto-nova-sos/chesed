import { useNavigate } from 'react-router-dom';
import { Alert } from '../ui/Alert';
import { Button } from '../ui/Button';
import type { DuplicateCheckResult } from '../../types';

interface DuplicateWarningProps {
  result: DuplicateCheckResult;
  onDismiss: () => void;
}

export function DuplicateWarning({ result, onDismiss }: DuplicateWarningProps) {
  const navigate = useNavigate();

  if (!result.has_duplicates) return null;

  return (
    <Alert variant="warning" onClose={onDismiss}>
      <div>
        <p className="font-medium">Possível duplicata encontrada!</p>
        <ul className="mt-2 space-y-1">
          {result.matches.map((match) => (
            <li key={match.id} className="flex items-center gap-2 text-sm">
              <span>
                {match.full_name} — {match.document_number}
              </span>
              <button
                type="button"
                onClick={() => navigate(`/persons/${match.id}`)}
                className="text-blue-600 underline hover:text-blue-800"
              >
                Ver
              </button>
            </li>
          ))}
        </ul>
        <div className="mt-3">
          <Button size="sm" variant="secondary" onClick={onDismiss}>
            Continuar mesmo assim
          </Button>
        </div>
      </div>
    </Alert>
  );
}
