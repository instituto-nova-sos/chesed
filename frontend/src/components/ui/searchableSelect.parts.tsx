import type { SearchableSelectOption } from './searchableSelect.utils';

interface CustomValueInputProps {
  label: string;
  placeholder: string;
  error?: string;
  onChange: (value: string) => void;
  onBackToList: () => void;
}

export function CustomValueInput({
  label,
  placeholder,
  error,
  onChange,
  onBackToList,
}: CustomValueInputProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      <input
        type="text"
        placeholder={placeholder}
        className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
          error ? 'border-red-300' : 'border-gray-300'
        }`}
        onChange={(e) => onChange(e.target.value || 'OTH')}
      />
      <button
        type="button"
        onClick={onBackToList}
        className="mt-1 text-xs text-blue-600 hover:underline"
      >
        Escolher da lista
      </button>
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </div>
  );
}

interface OptionsListProps {
  listRef: React.RefObject<HTMLUListElement | null>;
  filtered: SearchableSelectOption[];
  value: string;
  highlightedIndex: number;
  allowCustom: boolean;
  onHighlight: (index: number) => void;
  onSelect: (opt: SearchableSelectOption) => void;
  onCustom: () => void;
}

function optionClasses(idx: number, highlightedIndex: number, isSelected: boolean): string {
  if (idx === highlightedIndex) return 'bg-blue-50 text-blue-700';
  if (isSelected) return 'bg-gray-50 font-medium';
  return 'hover:bg-gray-50';
}

export function OptionsList({
  listRef,
  filtered,
  value,
  highlightedIndex,
  allowCustom,
  onHighlight,
  onSelect,
  onCustom,
}: OptionsListProps) {
  return (
    <ul
      ref={listRef}
      role="listbox"
      className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg"
    >
      {filtered.map((opt, idx) => (
        <li
          key={opt.value}
          role="option"
          aria-selected={opt.value === value}
          className={`cursor-pointer px-3 py-2 text-sm ${optionClasses(idx, highlightedIndex, opt.value === value)}`}
          onMouseEnter={() => onHighlight(idx)}
          onMouseDown={(e) => {
            e.preventDefault();
            onSelect(opt);
          }}
        >
          {opt.label}
        </li>
      ))}
      {allowCustom && (
        <li
          role="option"
          aria-selected={false}
          className={`cursor-pointer border-t px-3 py-2 text-sm italic text-gray-500 ${
            highlightedIndex === filtered.length ? 'bg-blue-50 text-blue-700' : 'hover:bg-gray-50'
          }`}
          onMouseEnter={() => onHighlight(filtered.length)}
          onMouseDown={(e) => {
            e.preventDefault();
            onCustom();
          }}
        >
          Outro (digitar manualmente)
        </li>
      )}
      {filtered.length === 0 && !allowCustom && (
        <li className="px-3 py-2 text-sm text-gray-400">
          Nenhum resultado encontrado
        </li>
      )}
    </ul>
  );
}
