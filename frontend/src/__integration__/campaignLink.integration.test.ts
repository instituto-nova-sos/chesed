/**
 * Integration test: campaign link contract on triage/attendance creation (S07.6).
 *
 * Pins the network-boundary contract: when a create input links a campaign,
 * the POST body carries `campaign_id`; when it does not, the key is absent
 * from the wire payload (the backend validates `omitempty,uuid`, so an empty
 * string would be rejected — absence is the contract for "no campaign").
 */

import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';

import { createTriage } from '../api/triages';
import { createAttendance } from '../api/attendances';
import type { CreateAttendanceInput, CreateTriageInput } from '../types';
import { API_BASE, server } from './server';
import './setup';

const CAMPAIGN_ID = '00000000-0000-0000-0000-00000000c001';

function captureTriagePost(captured: { body?: Record<string, unknown> }) {
  return http.post(`${API_BASE}/triages`, async ({ request }) => {
    captured.body = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({ id: 'server-triage-1', ...captured.body }, { status: 201 });
  });
}

function captureAttendancePost(captured: { body?: Record<string, unknown> }) {
  return http.post(`${API_BASE}/attendances`, async ({ request }) => {
    captured.body = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json(
      { id: 'server-attendance-1', status: 'SCHEDULED', ...captured.body },
      { status: 201 },
    );
  });
}

describe('integration: campaign link on create', () => {
  it('POST /triages carries campaign_id when the input links a campaign', async () => {
    const captured: { body?: Record<string, unknown> } = {};
    server.use(captureTriagePost(captured));

    const input: CreateTriageInput = {
      person_id: '00000000-0000-0000-0000-000000000001',
      main_complaint: 'Dor de cabeça',
      campaign_id: CAMPAIGN_ID,
    };
    const created = await createTriage(input);

    expect(created.id).toBe('server-triage-1');
    expect(captured.body?.campaign_id).toBe(CAMPAIGN_ID);
  });

  it('POST /triages omits the campaign_id key when no campaign is linked', async () => {
    const captured: { body?: Record<string, unknown> } = {};
    server.use(captureTriagePost(captured));

    await createTriage({
      person_id: '00000000-0000-0000-0000-000000000001',
      main_complaint: 'Dor de cabeça',
    });

    expect(captured.body).toBeDefined();
    expect('campaign_id' in (captured.body ?? {})).toBe(false);
  });

  it('POST /attendances carries campaign_id when the input links a campaign', async () => {
    const captured: { body?: Record<string, unknown> } = {};
    server.use(captureAttendancePost(captured));

    const input: CreateAttendanceInput = {
      person_id: '00000000-0000-0000-0000-000000000001',
      service_type_id: '00000000-0000-0000-0000-000000000010',
      professional_id: '00000000-0000-0000-0000-000000000020',
      campaign_id: CAMPAIGN_ID,
    };
    const created = await createAttendance(input);

    expect(created.id).toBe('server-attendance-1');
    expect(captured.body?.campaign_id).toBe(CAMPAIGN_ID);
  });

  it('POST /attendances omits the campaign_id key when no campaign is linked', async () => {
    const captured: { body?: Record<string, unknown> } = {};
    server.use(captureAttendancePost(captured));

    await createAttendance({
      person_id: '00000000-0000-0000-0000-000000000001',
      service_type_id: '00000000-0000-0000-0000-000000000010',
      professional_id: '00000000-0000-0000-0000-000000000020',
    });

    expect(captured.body).toBeDefined();
    expect('campaign_id' in (captured.body ?? {})).toBe(false);
  });
});
