import { useState } from 'react';
import { Button } from '../ui/Button';
import { Select } from '../ui/Select';
import { Input } from '../ui/Input';
import { Alert } from '../ui/Alert';
import { addPersonRole } from '../../api/persons';

interface AddRoleModalProps {
  personId: string;
  onClose: () => void;
  onAdded: () => void;
}

const roleOptions = [
  { value: 'VOLUNTEER', label: 'Voluntário' },
  { value: 'ASSISTED', label: 'Assistido' },
  { value: 'PROFESSIONAL', label: 'Profissional' },
  { value: 'COORDINATOR', label: 'Coordenador' },
  { value: 'ADMIN', label: 'Administrador' },
];

export function AddRoleModal({ personId, onClose, onAdded }: AddRoleModalProps) {
  const [roleType, setRoleType] = useState('VOLUNTEER');
  const [specialty, setSpecialty] = useState('');
  const [notes, setNotes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    try {
      await addPersonRole(personId, {
        role_type: roleType,
        professional_specialty: specialty || undefined,
        notes: notes || undefined,
      });
      onAdded();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao adicionar papel');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-medium text-gray-900">Adicionar Papel</h3>

        {error && (
          <Alert variant="error">
            {error}
          </Alert>
        )}

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <Select
            label="Tipo de Papel *"
            options={roleOptions}
            value={roleType}
            onChange={(e) => setRoleType(e.target.value)}
          />

          {roleType === 'PROFESSIONAL' && (
            <Input
              label="Especialidade"
              value={specialty}
              onChange={(e) => setSpecialty(e.target.value)}
              placeholder="Ex: Advocacia, Medicina, Psicologia"
            />
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Observações
            </label>
            <textarea
              className="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={2}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancelar
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              Adicionar
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
