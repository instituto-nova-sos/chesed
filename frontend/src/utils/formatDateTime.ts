const DATE_TIME_OPTIONS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
};

const DATE_OPTIONS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
};

/**
 * Formats an ISO timestamp as a pt-BR date + time. When `timeZone` is supplied
 * (the campus IANA zone), the wall-clock is rendered in that zone; otherwise it
 * falls back to the browser's local zone.
 */
export function formatDateTime(iso: string, timeZone?: string): string {
  const options = timeZone ? { ...DATE_TIME_OPTIONS, timeZone } : DATE_TIME_OPTIONS;
  return new Intl.DateTimeFormat('pt-BR', options).format(new Date(iso));
}

/**
 * Formats an ISO timestamp as a pt-BR calendar date. Defaults to UTC so a
 * date-only value never shifts across the day boundary; pass `timeZone` to
 * render the date in a specific zone instead.
 */
export function formatDate(iso: string, timeZone: string = 'UTC'): string {
  return new Intl.DateTimeFormat('pt-BR', { ...DATE_OPTIONS, timeZone }).format(new Date(iso));
}
