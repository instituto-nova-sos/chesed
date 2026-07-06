/**
 * Integration test: LGPD compliance report + donation receipt API surfaces
 * against MSW (Sprint 10, S11.4 / S11.5, docs/11-api-design.md).
 *
 * Exercises the real `apiClient` path: the compliance report query-string
 * contract, the CSV export request (bearer + format=csv), ApiError mapping on
 * a server error, and the donation receipt presigned-URL contract.
 */

import { http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import {
  getComplianceReport,
  downloadComplianceCSV,
} from '../api/compliance';
import { downloadReceipt } from '../api/donations';
import { ApiError } from '../api/client';
import type { ComplianceReport } from '../types/compliance';
import { API_BASE, server } from './server';
import './setup';

const DONATION_ID = '00000000-0000-0000-0000-0000000000d1';

function complianceFixture(
  overrides: Partial<ComplianceReport> = {},
): ComplianceReport {
  return {
    period: { start: '2020-01-01', end: '2030-12-31' },
    consents_by_type: { DATA_PROCESSING: 12, IMAGE_USAGE: 4 },
    active_consents: 14,
    revoked_consents: 2,
    anonymized_subjects: 1,
    data_subjects: 40,
    documents_stored: 9,
    ...overrides,
  };
}

describe('integration: compliance report + MSW', () => {
  it('requests the compliance report with the start/end query string', async () => {
    let seenUrl: URL | null = null;
    server.use(
      http.get(`${API_BASE}/reports/compliance`, ({ request }) => {
        seenUrl = new URL(request.url);
        return HttpResponse.json(complianceFixture());
      }),
    );

    const report = await getComplianceReport({
      start: '2020-01-01',
      end: '2030-12-31',
    });

    expect(seenUrl).not.toBeNull();
    expect(seenUrl!.searchParams.get('start')).toBe('2020-01-01');
    expect(seenUrl!.searchParams.get('end')).toBe('2030-12-31');
    expect(report.active_consents).toBe(14);
    expect(report.consents_by_type.DATA_PROCESSING).toBe(12);
  });

  it('maps a server error to ApiError', async () => {
    server.use(
      http.get(`${API_BASE}/reports/compliance`, () =>
        HttpResponse.json({ error: 'internal_error' }, { status: 500 }),
      ),
    );

    await expect(
      getComplianceReport({ start: '2020-01-01', end: '2030-12-31' }),
    ).rejects.toBeInstanceOf(ApiError);
  });

  it('downloads the CSV export with format=csv and returns a Blob', async () => {
    let seenUrl: URL | null = null;
    server.use(
      http.get(`${API_BASE}/reports/compliance/export`, ({ request }) => {
        seenUrl = new URL(request.url);
        return new HttpResponse('metric,value\nactive_consents,14\n', {
          headers: { 'Content-Type': 'text/csv' },
        });
      }),
    );

    const blob = await downloadComplianceCSV({
      start: '2020-01-01',
      end: '2030-12-31',
    });

    expect(seenUrl!.searchParams.get('format')).toBe('csv');
    expect(blob).toBeInstanceOf(Blob);
  });
});

describe('integration: donation receipt + MSW', () => {
  it('returns the presigned receipt url', async () => {
    server.use(
      http.get(`${API_BASE}/donations/${DONATION_ID}/receipt`, () =>
        HttpResponse.json({
          url: 'https://storage.example.com/receipts/x.pdf?sig=abc',
          expires_at: '2026-07-06T12:45:00Z',
        }),
      ),
    );

    const download = await downloadReceipt(DONATION_ID);
    expect(download.url).toContain('receipts/');
    expect(download.expires_at).toBe('2026-07-06T12:45:00Z');
  });

  it('maps a receipt server error to ApiError', async () => {
    server.use(
      http.get(`${API_BASE}/donations/${DONATION_ID}/receipt`, () =>
        HttpResponse.json({ error: 'not_found' }, { status: 404 }),
      ),
    );

    await expect(downloadReceipt(DONATION_ID)).rejects.toBeInstanceOf(ApiError);
  });
});
