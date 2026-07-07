import { useCallback, useMemo, useRef, useState } from 'react';
import { CustomValueInput, OptionsList } from './searchableSelect.parts';
import {
  normalizeSearchText as normalize,
  useClickOutside,
  useScrollHighlightedIntoView,
  type SearchableSelectOption,
} from './searchableSelect.utils';

interface SearchableSelectProps {
  label: string;
  options: SearchableSelectOption[];
  value: string;
  onChange: (value: string) => void;
  error?: string;
  placeholder?: string;
  allowCustom?: boolean;
  customPlaceholder?: string;
  /** Notifies the parent of the raw search text so it can drive a
   *  server-side search (options are still filtered client-side). */
  onSearchTextChange?: (text: string) => void;
}

interface KeyboardConfig {
  isOpen: boolean;
  open: () => void;
  close: () => void;
  itemCount: number;
  setHighlightedIndex: React.Dispatch<React.SetStateAction<number>>;
  confirm: () => void;
}

function useSelectKeyboard({
  isOpen,
  open,
  close,
  itemCount,
  setHighlightedIndex,
  confirm,
}: KeyboardConfig) {
  const handleOpenKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const actions: Record<string, () => void> = {
        ArrowDown: () => setHighlightedIndex((i) => (i + 1) % itemCount),
        ArrowUp: () => setHighlightedIndex((i) => (i - 1 + itemCount) % itemCount),
        Enter: confirm,
        Escape: close,
      };
      const action = actions[e.key];
      if (action) {
        e.preventDefault();
        action();
      }
    },
    [itemCount, confirm, close, setHighlightedIndex],
  );

  return useCallback(
    (e: React.KeyboardEvent) => {
      if (isOpen) {
        handleOpenKeyDown(e);
      } else if (e.key === 'ArrowDown' || e.key === 'Enter') {
        e.preventDefault();
        open();
      }
    },
    [isOpen, open, handleOpenKeyDown],
  );
}

function useSearchableSelect(
  options: SearchableSelectOption[],
  value: string,
  onChange: (value: string) => void,
  allowCustom: boolean,
) {
  const [isOpen, setIsOpen] = useState(false);
  const [isCustom, setIsCustom] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [highlightedIndex, setHighlightedIndex] = useState(0);

  const selectedOption = useMemo(
    () => options.find((o) => o.value === value),
    [options, value],
  );

  const filtered = useMemo(() => {
    if (!searchText) return options;
    const norm = normalize(searchText);
    return options.filter(
      (o) =>
        normalize(o.label).includes(norm) ||
        (o.searchTerms && normalize(o.searchTerms).includes(norm)),
    );
  }, [options, searchText]);

  const open = useCallback(() => setIsOpen(true), []);
  const close = useCallback(() => {
    setIsOpen(false);
    setSearchText('');
  }, []);

  const selectOption = useCallback(
    (opt: SearchableSelectOption) => {
      onChange(opt.value);
      close();
      setIsCustom(false);
    },
    [onChange, close],
  );

  const enterCustom = useCallback(() => {
    setIsCustom(true);
    close();
    onChange('OTH');
  }, [onChange, close]);

  const confirm = useCallback(() => {
    const highlighted = filtered[highlightedIndex];
    if (highlighted) selectOption(highlighted);
    else if (allowCustom) enterCustom();
  }, [filtered, highlightedIndex, allowCustom, selectOption, enterCustom]);

  const handleKeyDown = useSelectKeyboard({
    isOpen,
    open,
    close,
    itemCount: filtered.length + (allowCustom ? 1 : 0),
    setHighlightedIndex,
    confirm,
  });

  return {
    isOpen,
    setIsOpen,
    isCustom,
    setIsCustom,
    searchText,
    setSearchText,
    highlightedIndex,
    setHighlightedIndex,
    displayText: isOpen ? searchText : (selectedOption?.label ?? ''),
    filtered,
    close,
    selectOption,
    enterCustom,
    handleKeyDown,
  };
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
  onSearchTextChange,
}: SearchableSelectProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const s = useSearchableSelect(options, value, onChange, allowCustom);

  useClickOutside(containerRef, s.close);
  useScrollHighlightedIntoView(listRef, s.highlightedIndex, s.isOpen);

  if (s.isCustom) {
    return (
      <CustomValueInput
        label={label}
        placeholder={customPlaceholder}
        error={error}
        onChange={onChange}
        onBackToList={() => {
          s.setIsCustom(false);
          onChange('BRA');
        }}
      />
    );
  }

  return (
    <div ref={containerRef} className="relative">
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      <input
        type="text"
        role="combobox"
        aria-expanded={s.isOpen}
        aria-haspopup="listbox"
        value={s.displayText}
        placeholder={placeholder}
        className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
          error ? 'border-red-300' : 'border-gray-300'
        }`}
        onFocus={() => {
          s.setIsOpen(true);
          s.setSearchText('');
          s.setHighlightedIndex(0);
        }}
        onChange={(e) => {
          s.setSearchText(e.target.value);
          onSearchTextChange?.(e.target.value);
          s.setHighlightedIndex(0);
          if (!s.isOpen) s.setIsOpen(true);
        }}
        onKeyDown={s.handleKeyDown}
      />

      {s.isOpen && (
        <OptionsList
          listRef={listRef}
          filtered={s.filtered}
          value={value}
          highlightedIndex={s.highlightedIndex}
          allowCustom={allowCustom}
          onHighlight={s.setHighlightedIndex}
          onSelect={s.selectOption}
          onCustom={s.enterCustom}
        />
      )}
      {error && <p className="mt-1 text-xs text-red-600">{error}</p>}
    </div>
  );
}
