import type { FormEvent } from 'react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Select } from '../ui/Select';

export interface AuditFilterValues {
  start: string;
  end: string;
  user_id: string;
  entity_type: string;
  action_type: string;
}

const ACTION_OPTIONS = [
  'CREATE',
  'READ',
  'UPDATE',
  'DELETE',
  'LOGIN',
  'LOGOUT',
  'EXPORT',
  'PERMISSION_CHANGE',
  'ACCESS_DENIED',
].map((value) => ({ value, label: value }));

interface AuditLogFiltersProps {
  values: AuditFilterValues;
  onChange: (key: keyof AuditFilterValues, value: string) => void;
  onSubmit: (event: FormEvent) => void;
  isLoading: boolean;
}

export function AuditLogFilters({
  values,
  onChange,
  onSubmit,
  isLoading,
}: AuditLogFiltersProps) {
  return (
    <form
      onSubmit={onSubmit}
      className="grid grid-cols-1 gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm sm:grid-cols-2 lg:grid-cols-3"
    >
      <Input
        id="audit-start"
        label="Início"
        type="date"
        value={values.start}
        onChange={(e) => onChange('start', e.target.value)}
        required
      />
      <Input
        id="audit-end"
        label="Fim"
        type="date"
        value={values.end}
        onChange={(e) => onChange('end', e.target.value)}
        required
      />
      <Input
        id="audit-entity"
        label="Entidade"
        placeholder="person, donation…"
        value={values.entity_type}
        onChange={(e) => onChange('entity_type', e.target.value)}
      />
      <Select
        id="audit-action"
        label="Ação"
        placeholder="Todas"
        options={ACTION_OPTIONS}
        value={values.action_type}
        onChange={(e) => onChange('action_type', e.target.value)}
      />
      <Input
        id="audit-user"
        label="ID do usuário"
        placeholder="UUID"
        value={values.user_id}
        onChange={(e) => onChange('user_id', e.target.value)}
      />
      <div className="flex items-end">
        <Button type="submit" isLoading={isLoading}>
          Buscar
        </Button>
      </div>
    </form>
  );
}
