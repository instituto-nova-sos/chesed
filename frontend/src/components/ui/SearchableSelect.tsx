import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

interface SearchableSelectOption {
  value: string;
  label: string;
  searchTerms?: string;
}

interface SearchableSelectProps {
  label: string;
  options: SearchableSelectOption[];
  value: string;
  onChange: (value: string) => void;
  error?: string;
  placeholder?: string;
  allowCustom?: boolean;
  customPlaceholder?: string;
}

function normalize(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase();
}

export function SearchableSelect({
  label,
  options,
  value,
  onChange,
  error,
  placeholder = 'Buscar...',
  allowCustom = false,
  customPlaceholder = 'Digite manualmente...',
}: SearchableSelectProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const [isOpen, setIsOpen] = useState(false);
  const [isCustom, setIsCustom] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [highlightedIndex, setHighlightedIndex] = useState(0);

  const selectedOption = useMemo(
    () => options.find((o) => o.value === value),
    [options, value],
  );

  const displayText = isOpen ? searchText : (selectedOption?.label ?? '');

  const filtered = useMemo(() => {
    if (!searchText) return options;
    const norm = normalize(searchText);
    return options.filter(
      (o) =>
        normalize(o.label).includes(norm) ||
        (o.searchTerms && normalize(o.searchTerms).includes(norm)),
    );
  }, [options, searchText]);

  // Click outside to close
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearchText('');
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  // Scroll highlighted item into view
  useEffect(() => {
    if (!isOpen || !listRef.current) return;
    const item = listRef.current.children[highlightedIndex] as HTMLElement | undefined;
    item?.scrollIntoView({ block: 'nearest' });
  }, [highlightedIndex, isOpen]);

  const selectOption = useCallback(
    (opt: SearchableSelectOption) => {
      onChange(opt.value);
      setIsOpen(false);
      setSearchText('');
      setIsCustom(false);
    },
    [onChange],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!isOpen) {
        if (e.key === 'ArrowDown' || e.key === 'Enter') {
          e.preventDefault();
          setIsOpen(true);
        }
        return;
      }

      const totalItems = filtered.length + (allowCustom ? 1 : 0);

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setHighlightedIndex((i) => (i + 1) % totalItems);
          break;
        case 'ArrowUp':
          e.preventDefault();
          setHighlightedIndex((i) => (i - 1 + totalItems) % totalItems);
          break;
        case 'Enter':
          e.preventDefault();
          if (highlightedIndex < filtered.length) {
            selectOption(filtered[highlightedIndex]);
          } else if (allowCustom) {
            setIsCustom(true);
            setIsOpen(false);
            setSearchText('');
            onChange('OTH');
          }
          break;
        case 'Escape':
          e.preventDefault();
          setIsOpen(false);
          setSearchText('');
          break;
      }
    },
    [isOpen, filtered, highlightedIndex, allowCustom, selectOption, onChange],
  );

  if (isCustom) {
    return (
      <div>
        <label className="block text-sm font-medium text-gray-700">{label}</label>
        <input
          type="text"
          placeholder={customPlaceholder}
          className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
            error ? 'border-red-300' : 'border-gray-300'
          }`}
          onChange={(e) => onChange(e.target.value || 'OTH')}
        />
        <button
          type="button"
          onClick={() => {
            setIsCustom(false);
            onChange('BRA');
          }}
          className="mt-1 text-xs text-blue-600 hover:underline"
        >
          Escolher da lista
        </button>
        {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
      </div>
    );
  }

  return (
    <div ref={containerRef} className="relative">
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      <input
        ref={inputRef}
        type="text"
        role="combobox"
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        value={displayText}
        placeholder={placeholder}
        className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
          error ? 'border-red-300' : 'border-gray-300'
        }`}
        onFocus={() => {
          setIsOpen(true);
          setSearchText('');
          setHighlightedIndex(0);
        }}
        onChange={(e) => {
          setSearchText(e.target.value);
          setHighlightedIndex(0);
          if (!isOpen) setIsOpen(true);
        }}
        onKeyDown={handleKeyDown}
      />

      {isOpen && (
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
              className={`cursor-pointer px-3 py-2 text-sm ${
                idx === highlightedIndex
                  ? 'bg-blue-50 text-blue-700'
                  : opt.value === value
                    ? 'bg-gray-50 font-medium'
                    : 'hover:bg-gray-50'
              }`}
              onMouseEnter={() => setHighlightedIndex(idx)}
              onMouseDown={(e) => {
                e.preventDefault();
                selectOption(opt);
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
              onMouseEnter={() => setHighlightedIndex(filtered.length)}
              onMouseDown={(e) => {
                e.preventDefault();
                setIsCustom(true);
                setIsOpen(false);
                setSearchText('');
                onChange('OTH');
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
      )}
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </div>
  );
}
