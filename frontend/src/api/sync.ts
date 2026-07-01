import { apiClient } from './client';
import type {
  SyncPushRecord,
  SyncPushResponse,
  PullRecord,
} from '../offline/syncEngine';

export interface SyncPullResponse {
  records: PullRecord[];
  server_timestamp: string;
  has_more: boolean;
  next_since?: string;
}

/** syncPush uploads a batch of offline records to POST /sync/push. */
export function syncPush(records: SyncPushRecord[]): Promise<SyncPushResponse> {
  return apiClient<SyncPushResponse>('/sync/push', {
    method: 'POST',
    body: JSON.stringify({ records }),
  });
}

/**
 * syncPull fetches records updated since the given cursor from GET /sync/pull.
 * `since` is required by the backend; `entityTypes` narrows the pull when set.
 */
export function syncPull(
  since: string,
  entityTypes?: string[],
  limit?: number,
): Promise<SyncPullResponse> {
  const params = new URLSearchParams({ since });
  if (entityTypes?.length) params.set('entity_types', entityTypes.join(','));
  if (limit) params.set('limit', String(limit));
  return apiClient<SyncPullResponse>(`/sync/pull?${params.toString()}`);
}
