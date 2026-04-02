# 12 - Offline Sync Strategy

## Overview

The system must support field operations during social action events where internet connectivity is unreliable or unavailable. Volunteers and professionals need to register persons, record triages, and log attendances on mobile devices without internet, then synchronize when connectivity returns.

---

## Architecture

```
┌──────────────────────────────────┐
│          Mobile Device            │
│                                   │
│  ┌─────────────┐  ┌───────────┐ │
│  │  React PWA   │  │  Service  │ │
│  │  (UI Layer)  │  │  Worker   │ │
│  └──────┬──────┘  └─────┬─────┘ │
│         │                │       │
│  ┌──────┴────────────────┴────┐  │
│  │      Dexie.js (IndexedDB)   │ │
│  │  ┌────────┐ ┌────────────┐  │ │
│  │  │ Data   │ │ Sync Queue │  │ │
│  │  │ Store  │ │            │  │ │
│  │  └────────┘ └────────────┘  │ │
│  └─────────────┬──────────────┘  │
│                │                  │
└────────────────┼──────────────────┘
                 │ (when online)
                 ▼
┌──────────────────────────────────┐
│          Go API Server            │
│  ┌────────────────────────────┐  │
│  │     Sync Engine             │  │
│  │  - Validate & persist       │  │
│  │  - Detect conflicts         │  │
│  │  - Return server state      │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

---

## Local Storage (IndexedDB via Dexie.js)

### Database Schema

```typescript
import Dexie, { Table } from 'dexie';

interface LocalPerson {
  id: string;            // UUID (client-generated)
  syncId: string;        // Same as id for offline-created records
  data: PersonData;      // Full person object
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

interface LocalTriage {
  id: string;
  syncId: string;
  data: TriageData;
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

interface LocalAttendance {
  id: string;
  syncId: string;
  data: AttendanceData;
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;
  serverUpdatedAt?: string;
}

interface SyncQueueItem {
  id?: number;           // Auto-increment
  entityType: 'person' | 'triage' | 'attendance';
  entityId: string;
  action: 'create' | 'update';
  data: any;
  createdAt: string;
  retryCount: number;
  lastError?: string;
}

interface SyncMeta {
  key: string;
  value: string;
}

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

### What Gets Stored Offline

| Entity | Create Offline | Update Offline | Read Offline |
|--------|---------------|---------------|-------------|
| Person | Yes | Yes (basic fields) | Yes (cached) |
| Triage | Yes | No | Yes (cached) |
| Attendance | Yes | Yes (status, observations) | Yes (cached) |
| Service Types | No (reference data) | No | Yes (pre-cached) |
| Campaigns | No | No | Yes (pre-cached) |

### Pre-caching Strategy

When the app first connects (or when explicitly triggered):
1. Download all service types → IndexedDB
2. Download active campaigns → IndexedDB
3. Download recent persons for the user's campus → IndexedDB (last 30 days)

This enables person search and form population while offline.

---

## Sync Protocol

### Push: Client → Server

When the device comes online (or user taps "Sync Now"):

1. Read all items from `syncQueue` ordered by `createdAt`
2. Batch into groups of 50 records
3. Send `POST /api/v1/sync/push` with batch
4. Process server response:
   - `created`: Mark local record as synced; store server-assigned metadata
   - `conflict`: Mark local record as conflict; store both versions
   - `error`: Increment retry count; leave in queue
5. Remove successfully synced items from queue
6. Retry failed items (max 5 retries; exponential backoff)

```
Client                          Server
  │                               │
  │  POST /sync/push              │
  │  [person, triage, attendance] │
  │──────────────────────────────>│
  │                               │ Validate each record
  │                               │ Check for conflicts
  │                               │ Persist valid records
  │  Response                     │
  │  [created, created, conflict] │
  │<──────────────────────────────│
  │                               │
  │  Update local IndexedDB       │
  │  Remove from sync queue       │
```

### Pull: Server → Client

After push completes (or periodically):

1. Read `lastSyncTimestamp` from `syncMeta`
2. Send `GET /api/v1/sync/pull?since=<timestamp>&entity_types=person,triage,attendance`
3. Merge received records into IndexedDB:
   - New records: Insert
   - Updated records: Replace if server version is newer
   - Deleted records: Mark as inactive locally
4. Update `lastSyncTimestamp` with server timestamp from response
5. If `has_more` is true, repeat with the returned timestamp

---

## Conflict Resolution Strategy

### MVP: Last-Write-Wins (LWW)

For the MVP, the simplest approach:

1. **Same record modified on client and server**: Server version wins. Client changes are stored in a `conflicts` table for manual review.
2. **Duplicate detection**: If a person with the same document_number already exists on the server, the push returns `conflict` with the existing record's ID.
3. **Idempotency**: `sync_id` (UUID) is used as an idempotency key. Pushing the same `sync_id` twice results in a no-op on the second push.

### Conflict Detection

```
Push request arrives:
  1. Check sync_id → already processed? → Return existing result (idempotent)
  2. Check entity constraints:
     - Person: document_number unique?
     - Attendance: valid status transition?
  3. Check timestamp:
     - Record modified on server since client's last sync?
     - Yes → Conflict
     - No → Accept
```

### Conflict Storage

```typescript
interface ConflictRecord {
  id: string;
  entityType: string;
  entityId: string;
  clientVersion: any;
  serverVersion: any;
  detectedAt: string;
  resolvedAt?: string;
  resolvedBy?: string;
  resolution?: 'keep_client' | 'keep_server' | 'merged';
}
```

### Phase 3: Manual Conflict Resolution UI

For Phase 3, add a conflict resolution interface:
- Show side-by-side comparison of client vs. server version
- Allow user to pick one or merge fields manually
- Log resolution in audit trail

---

## Retry and Idempotency

### Retry Strategy

```
Attempt 1: Immediate
Attempt 2: 5 seconds delay
Attempt 3: 30 seconds delay
Attempt 4: 2 minutes delay
Attempt 5: 10 minutes delay
After 5 failures: Stop retrying; flag for manual review
```

### Idempotency

Every offline-created record has a `sync_id` (UUID generated on the client). This serves as an idempotency key:

- Client generates UUID before saving to IndexedDB
- UUID is sent with the push request
- Server checks if `sync_id` already exists
- If yes: Returns the existing record (no duplicate creation)
- If no: Creates the record with the provided `sync_id`

This means network failures during push are safe — the client can retry without creating duplicates.

---

## UUID Generation for Offline Records

All records created offline use **UUIDv4** generated on the client:

```typescript
// Using crypto.randomUUID() (available in all modern browsers)
const id = crypto.randomUUID();
```

This eliminates the need for server-assigned auto-increment IDs and prevents ID collisions between devices.

---

## Connectivity Detection

```typescript
// Online/offline detection
const isOnline = (): boolean => navigator.onLine;

// Listen for connectivity changes
window.addEventListener('online', () => {
  // Trigger sync
  syncEngine.pushPendingRecords();
});

window.addEventListener('offline', () => {
  // Update UI to show offline mode
  setOfflineMode(true);
});

// Periodic check (navigator.onLine can be unreliable)
// Ping health endpoint every 30 seconds when "online"
const checkConnectivity = async (): Promise<boolean> => {
  try {
    const response = await fetch('/api/v1/health', {
      method: 'HEAD',
      cache: 'no-store'
    });
    return response.ok;
  } catch {
    return false;
  }
};
```

---

## Service Worker Caching

### Caching Strategy

| Resource | Strategy | Reason |
|----------|----------|--------|
| App shell (HTML, JS, CSS) | Cache-first, update in background | Fast load; update on next visit |
| API responses (GET) | Network-first, fall back to cache | Fresh data when online; cached when offline |
| API mutations (POST, PATCH) | Network-only, queue if offline | Mutations must reach server; queue for sync |
| Static assets (fonts, icons) | Cache-first | Rarely change |
| Images/documents | Cache-first with expiry | Large; don't re-download frequently |

### PWA Manifest

```json
{
  "name": "SOS Gestao - Instituto Nova SOS",
  "short_name": "SOS Gestao",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#1a56db",
  "icons": [
    { "src": "/icons/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/icon-512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

---

## Bandwidth Optimization

1. **Batch sync**: Send up to 50 records per push request (reduces HTTP overhead)
2. **Delta sync**: Pull endpoint returns only records modified since last sync
3. **Compression**: gzip on all API responses
4. **Pagination**: Pull results paginated to prevent memory issues on large datasets
5. **Selective sync**: Only sync entity types the user interacts with (based on role)

---

## Security Considerations

1. **Local data encryption**: Use the Web Crypto API to encrypt sensitive fields in IndexedDB (person names, CPF numbers) when the device supports it
2. **Token storage**: JWT refresh token stored in IndexedDB, encrypted
3. **Session expiry**: If refresh token expires while offline, user must re-authenticate when online
4. **Data wipe**: Provide "Clear local data" option for shared devices
5. **Audit**: Log sync events (push/pull) in the server audit trail

---

## Limitations and Trade-offs

| Trade-off | Decision | Rationale |
|-----------|----------|-----------|
| LWW may lose data | Accept for MVP | Conflicts are rare; flagged for review |
| Pre-cached data may be stale | Accept | Freshness check on next sync |
| IndexedDB size limits | Monitor | Browsers allow ~50MB+; sufficient for event data |
| No real-time updates | Accept | Polling on sync pull is sufficient |
| Shared device data leakage | Mitigate | Clear local data on logout; encrypt sensitive fields |
