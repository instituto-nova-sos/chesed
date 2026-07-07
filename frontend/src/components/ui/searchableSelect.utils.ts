import { useEffect } from 'react';

export interface SearchableSelectOption {
  value: string;
  label: string;
  searchTerms?: string;
}

export function normalizeSearchText(text: string): string {
  return text
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase();
}

export function useClickOutside(
  ref: React.RefObject<HTMLElement | null>,
  onOutside: () => void,
) {
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onOutside();
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [ref, onOutside]);
}

export function useScrollHighlightedIntoView(
  listRef: React.RefObject<HTMLUListElement | null>,
  index: number,
  active: boolean,
) {
  useEffect(() => {
    if (!active || !listRef.current) return;
    const item = listRef.current.children[index] as HTMLElement | undefined;
    item?.scrollIntoView?.({ block: 'nearest' });
  }, [listRef, index, active]);
}
