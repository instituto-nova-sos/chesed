import { describe, expect, it } from 'vitest';
import { formatDate, formatDateTime } from './formatDateTime';

// Fixed UTC instant: 2026-07-06 23:30 UTC.
// In America/Sao_Paulo (UTC-3) this is 20:30 the same day.
// In Europe/Lisbon (UTC+1 in July) this is 00:30 the next day.
const ISO = '2026-07-06T23:30:00Z';

describe('formatDateTime', () => {
  it('renders a different local wall-clock under two campus time zones', () => {
    const saoPaulo = formatDateTime(ISO, 'America/Sao_Paulo');
    const lisbon = formatDateTime(ISO, 'Europe/Lisbon');

    expect(saoPaulo).not.toBe(lisbon);
    expect(saoPaulo).toContain('20:30');
    expect(lisbon).toContain('00:30');
    // Lisbon crosses midnight into the 7th.
    expect(lisbon).toContain('07');
    expect(saoPaulo).toContain('06');
  });

  it('falls back to a valid string when no time zone is provided', () => {
    expect(formatDateTime(ISO)).toMatch(/\d{2}:\d{2}/);
  });
});

describe('formatDate', () => {
  it('renders a stable UTC calendar date when no time zone is passed', () => {
    expect(formatDate('2026-07-06T00:00:00Z')).toBe('06/07/2026');
  });

  it('does not shift the date backwards for an early-UTC instant', () => {
    // 2026-07-06T02:00Z would be 2026-07-05 in America/Sao_Paulo, but the
    // date-only formatter must pin UTC and keep the 6th.
    expect(formatDate('2026-07-06T02:00:00Z')).toBe('06/07/2026');
  });

  it('honours an explicit time zone when one is supplied', () => {
    expect(formatDate('2026-07-06T02:00:00Z', 'America/Sao_Paulo')).toBe('05/07/2026');
  });
});
