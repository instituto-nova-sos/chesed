# Skill: Design Offline Support

## Purpose

Design the offline sync strategy for a specific feature, including Dexie.js table schemas, sync queue entries, client UUID generation, sync status indicators, pre-caching, LWW conflict detection, and token refresh handling. Follows the architecture defined in `docs/12-offline-sync-strategy.md`.

## When to Use / Trigger

- When implementing a feature that must work offline (person creation, triage, attendance).
- When a user says "design offline support for feature X".
- After the frontend feature design identifies offline requirements.

## Role / Expertise

Offline-first PWA engineer with expertise in:
- IndexedDB via Dexie.js.
- Service Worker caching strategies.
- Sync queue patterns (outbox pattern).
- UUID generation for offline record creation.
- Conflict resolution (Last-Write-Wins).
- Token management with keycloak-js in offline scenarios.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Feature specification | Yes | Story analysis or frontend design |
| Offline sync strategy | Yes | `docs/12-offline-sync-strategy.md` |
| API sync endpoints | Yes | `docs/11-api-design.md` (sync push/pull) |
| Entity data shape | Yes | TypeScript interfaces from frontend design |

## Process

### Step 1: Determine Offline Capabilities

Reference the offline capability matrix from `docs/12-offline-sync-strategy.md`:

| Entity | Create Offline | Update Offline | Read Offline |
|--------|---------------|---------------|-------------|
| Person | Yes | Yes (basic fields) | Yes (cached) |
| Triage | Yes | No | Yes (cached) |
| Attendance | Yes | Yes (status, observations) | Yes (cached) |
| Service Types | No (reference data) | No | Yes (pre-cached) |

Determine which capabilities the feature needs.

### Step 2: Define Dexie.js Table Schema

Location: `frontend/src/offline/db.ts`

The project uses a single Dexie database named `sos-gestao` with these stores:

```typescript
class OfflineDB extends Dexie {
  persons!: Table<LocalPerson>;
  triages!: Table<LocalTriage>;
  attendances!: Table<LocalAttendance>;
  syncQueue!: Table<SyncQueueItem>;
  syncMeta!: Table<SyncMeta>;

  constructor() {
    super('sos-gestao');
    this.version(1).stores({
      persons: 'id, syncStatus, [data.document_number]',
      triages: 'id, syncStatus, [data.person_id]',
      attendances: 'id, syncStatus, [data.person_id], [data.status]',
      syncQueue: '++id, entityType, entityId, createdAt',
      syncMeta: 'key'
    });
  }
}
```

For the feature, define the local record structure:

```typescript
interface LocalRecord<T> {
  id: string;            // UUID (client-generated via crypto.randomUUID())
  syncId: string;        // Same as id for offline-created records
  data: T;               // Full entity data matching API request shape
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string; // ISO timestamp
  serverUpdatedAt?: string;
}
```

### Step 3: Define Client UUID Generation

For offline-created records, UUIDs must be generated client-side to avoid server round-trips:

```typescript
// frontend/src/utils/uuid.ts
export function generateUUID(): string {
  return crypto.randomUUID(); // Web Crypto API, supported in modern browsers
}
```

The generated UUID becomes both the `id` and the `sync_id` sent to the server. The server uses `sync_id` for idempotency (pushing the same sync_id twice is a no-op).

### Step 4: Define Sync Queue Entry Format

When a record is created or updated offline, an entry is added to the sync queue:

```typescript
interface SyncQueueItem {
  id?: number;           // Auto-increment (Dexie manages)
  entityType: 'person' | 'triage' | 'attendance';
  entityId: string;      // UUID of the affected record
  action: 'create' | 'update';
  data: unknown;         // Full payload matching API request body
  createdAt: string;     // ISO timestamp
  retryCount: number;    // Starts at 0, max 5
  lastError?: string;    // Last sync error message
}
```

Queue processing rules:
1. Process in FIFO order (by `createdAt`).
2. Batch into groups of 50.
3. Send via `POST /api/v1/sync/push`.
4. On success: remove from queue, mark local record as `synced`.
5. On conflict: mark local record as `conflict`, store both versions.
6. On error: increment `retryCount`, apply exponential backoff.
7. After 5 retries: flag for manual intervention.

### Step 5: Define Pre-caching Strategy

On app startup (when online), pre-cache reference data and recent records:

```typescript
async function preCacheData(keycloakToken: string): Promise<void> {
  // 1. Service types (reference data, rarely changes)
  const serviceTypes = await fetchServiceTypes(keycloakToken);
  await db.serviceTypes.bulkPut(serviceTypes);

  // 2. Recent persons for user's campus (last 30 days)
  const persons = await fetchRecentPersons(keycloakToken, { days: 30 });
  await db.persons.bulkPut(persons.map(p => ({
    id: p.id,
    syncId: p.id,
    data: p,
    syncStatus: 'synced' as const,
    localCreatedAt: p.created_at,
    serverUpdatedAt: p.updated_at,
  })));

  // 3. Update last sync timestamp
  await db.syncMeta.put({ key: 'lastSyncTimestamp', value: new Date().toISOString() });
}
```

### Step 6: Define Sync Status Indicator

UI must show sync status for each record and globally:

**Per-record indicator**:
```typescript
// Component: SyncBadge
// Props: syncStatus: 'pending' | 'synced' | 'conflict'
// Display:
//   pending  -> yellow badge "Pendente"
//   synced   -> green badge "Sincronizado"
//   conflict -> red badge "Conflito"
```

**Global indicator** (in navbar):
```typescript
// Component: SyncStatusIndicator
// Shows:
//   Online + no pending    -> green dot
//   Online + pending       -> yellow dot + count + "Sync Now" button
//   Offline                -> gray dot + "Offline"
//   Syncing                -> spinning icon + "Sincronizando..."
```

### Step 7: Define Conflict Resolution Flow

MVP uses Last-Write-Wins (LWW) per `docs/12-offline-sync-strategy.md`:

1. Push arrives at server.
2. Server checks `sync_id` for idempotency.
3. If record was modified on server since client's `serverUpdatedAt`, return `conflict`.
4. Client receives conflict: mark local record as `conflict`.
5. UI shows conflict badge with option to review.
6. For MVP, server version wins automatically. Client-side changes are preserved in a `conflicts` view for manual review.

### Step 8: Define Token Refresh Handling

Offline sync requires a valid Keycloak access token:

```typescript
async function syncWithTokenRefresh(): Promise<void> {
  // 1. Attempt silent token refresh via keycloak-js
  try {
    const refreshed = await keycloak.updateToken(30); // Refresh if expires within 30s
    if (refreshed) {
      // Token refreshed, proceed with sync
    }
  } catch (error) {
    // Refresh failed (offline or token expired beyond refresh window)
    // Defer sync until user re-authenticates
    return;
  }

  // 2. Process sync queue with fresh token
  await processSyncQueue(keycloak.token!);
}
```

For field workers: request `offline_access` scope from Keycloak to get 14-day refresh tokens.

### Step 9: Define IndexedDB Encryption (Sensitive Data)

Per threat T4 (offline device theft), sensitive fields must be encrypted:

```typescript
// Use Web Crypto API for encryption
// Key derived from user's Keycloak session
// Encrypted fields: document_number, phone, email, health data, social observations

async function encryptField(data: string, key: CryptoKey): Promise<string> {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encoded = new TextEncoder().encode(data);
  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encoded);
  // Return base64(iv + encrypted)
}
```

On logout, call `db.delete()` to wipe all local data.

## Outputs / Deliverables

1. **Dexie.js table schema** additions/modifications.
2. **Sync queue entry format** for the feature's entities.
3. **UUID generation** approach.
4. **Pre-caching plan**: what data to cache, when, how much.
5. **Sync flow**: push/pull sequence with error handling.
6. **UI components**: SyncBadge, SyncStatusIndicator specifications.
7. **Conflict resolution**: LWW implementation details.
8. **Token handling**: Refresh strategy for offline-to-online transition.
9. **Encryption**: Fields to encrypt, key management approach.
10. **Test cases**: Offline-specific scenarios to test.

## References

| Document | Path | Usage |
|----------|------|-------|
| Offline sync strategy | `docs/12-offline-sync-strategy.md` | Architecture and protocol |
| Threat model | `docs/18-threat-model.md` | T4 (device theft), T5 (sync injection) |
| API design | `docs/11-api-design.md` | Sync push/pull endpoints |
| Security | `docs/13-security-and-compliance.md` | Data classification for encryption |

## Constraints / Quality Bar

- Offline-created records MUST use client-generated UUIDs (crypto.randomUUID()).
- Sync queue MUST process in FIFO order.
- Max 5 retries with exponential backoff.
- Batch size: 50 records per sync push.
- Sensitive PII MUST be encrypted in IndexedDB.
- All local data MUST be wiped on logout.
- Token refresh MUST be attempted before sync; if it fails, defer sync.
- Pre-cached data limited to last 30 days for the user's campus.

## Interaction with Other Artifacts

- **Invoked by agents**: frontend-engineer.
- **Depends on skills**: design-frontend-feature (provides entity types and API shapes).
- **Feeds into skills**: design-test-plan (offline test scenarios), review-security (encryption review).
