# Playbook: Add Offline Support

## Purpose

Guide for adding offline capability to a Chesed feature, including IndexedDB storage, sync queue, encryption, and conflict handling. Reference: `docs/12-offline-sync-strategy.md`.

---

## Prerequisites

- The feature's API endpoints are implemented and documented in `docs/11-api-design.md`
- The sync endpoints exist: `POST /api/v1/sync/push`, `GET /api/v1/sync/pull`, `GET /api/v1/sync/status`
- The feature is confirmed as offline-capable in the MVP scope (`docs/07-mvp-scope.md`)

---

## Offline Classification

Before implementing, classify the feature into one of three categories:

| Classification | Description | Examples |
|---|---|---|
| **Fully offline-capable** | Create and read without network. Sync when online. | Person registration, triage creation |
| **Read-only offline** | View cached data offline. Creation requires network. | Person list, attendance list, service type catalog |
| **Online-only** | No offline support. Requires network. | Reports, user management, audit logs, CSV export |

The classification determines which steps below apply.

---

## Steps

### Step 1: Classify the Feature

Check `docs/07-mvp-scope.md` and `docs/12-offline-sync-strategy.md`:

- **Person registration**: Fully offline-capable (RF-46)
- **Triage creation**: Fully offline-capable (RF-47)
- **Attendance recording**: Fully offline-capable (RF-48)
- **Person search/list**: Read-only offline (cached data)
- **Service type catalog**: Read-only offline (cached on first load)
- **Reports**: Online-only
- **User management**: Online-only
- **Audit logs**: Online-only

If the feature is **online-only**, skip to Step 7 (show appropriate offline message).

### Step 2: Add Dexie Table

File: `frontend/src/offline/db.ts`

Add a new table to the Dexie database for the entity. Follow the schema from `docs/12-offline-sync-strategy.md`.

```typescript
import Dexie, { Table } from 'dexie';

// Entity-specific local storage interfaces
interface LocalPerson {
  id: string;            // UUID (client-generated for offline records)
  syncId: string;        // Same as id for offline-created; server id for synced records
  data: PersonData;      // Full entity payload matching API contract
  syncStatus: 'pending' | 'synced' | 'conflict';
  localCreatedAt: string;   // ISO 8601 timestamp of local creation
  serverUpdatedAt?: string; // ISO 8601 timestamp from server (after sync)
  encryptedFields?: string; // Encrypted PII blob (document_number, health data)
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

// Sync queue for outbound changes
interface SyncQueueItem {
  id?: number;               // Auto-increment
  entityType: 'person' | 'triage' | 'attendance';
  action: 'create' | 'update';
  syncId: string;            // UUID linking to the local entity record
  data: Record<string, unknown>;  // Payload to send to server
  timestamp: string;         // ISO 8601 of when the action occurred
  syncStatus: 'pending' | 'in_progress' | 'completed' | 'failed';
  retryCount: number;
  lastError?: string;
}

class ChesedDatabase extends Dexie {
  persons!: Table<LocalPerson, string>;
  triages!: Table<LocalTriage, string>;
  attendances!: Table<LocalAttendance, string>;
  syncQueue!: Table<SyncQueueItem, number>;

  constructor() {
    super('chesed');
    this.version(1).stores({
      persons: 'id, syncId, syncStatus, localCreatedAt',
      triages: 'id, syncId, syncStatus, localCreatedAt, [data.person_id]',
      attendances: 'id, syncId, syncStatus, localCreatedAt, [data.person_id]',
      syncQueue: '++id, entityType, syncStatus, timestamp',
    });
  }
}

export const db = new ChesedDatabase();
```

When adding a new entity type, increment the Dexie version number and add the new table in a version upgrade:

```typescript
this.version(2).stores({
  // Existing tables remain unchanged
  persons: 'id, syncId, syncStatus, localCreatedAt',
  // New table
  newEntity: 'id, syncId, syncStatus',
});
```

### Step 3: Generate UUID Client-Side

For offline-created records, generate UUIDs on the client using the Web Crypto API (not Math.random-based UUIDs):

```typescript
// Use the browser's built-in crypto.randomUUID()
const offlineId = crypto.randomUUID();
```

This UUID serves as both the local `id` and the `sync_id` sent to the server. The server uses `sync_id` for idempotency (if the same record is pushed twice, the server recognizes the duplicate).

### Step 4: Add Sync Queue Entry on Local Save

When the user saves data while offline (or while online as a performance optimization), write to both the local Dexie table and the sync queue:

```typescript
// frontend/src/offline/operations/personOperations.ts

import { db } from '../db';
import { encryptSensitiveFields } from '../encryption';
import type { CreatePersonRequest } from '../../types/person';

export async function createPersonOffline(data: CreatePersonRequest): Promise<string> {
  const id = crypto.randomUUID();
  const timestamp = new Date().toISOString();

  // Encrypt PII before storing in IndexedDB
  const encryptedFields = await encryptSensitiveFields({
    document_number: data.document_number,
  });

  // Store in local table
  await db.persons.add({
    id,
    syncId: id,
    data: { ...data, document_number: undefined }, // Remove PII from plain data
    syncStatus: 'pending',
    localCreatedAt: timestamp,
    encryptedFields,
  });

  // Add to sync queue
  await db.syncQueue.add({
    entityType: 'person',
    action: 'create',
    syncId: id,
    data: data as Record<string, unknown>,
    timestamp,
    syncStatus: 'pending',
    retryCount: 0,
  });

  return id;
}
```

### Step 5: Verify Sync Push Endpoint

Confirm that `POST /api/v1/sync/push` accepts the entity type. The request format from `docs/11-api-design.md`:

```json
{
  "device_id": "uuid",
  "records": [
    {
      "entity_type": "person",
      "sync_id": "uuid",
      "data": { "full_name": "...", "document_type": "CPF", ... },
      "created_at": "2026-04-02T10:30:00Z"
    }
  ]
}
```

The sync push handler on the backend must:
- Validate the payload with the same rules as the direct API endpoint
- Check `sync_id` for idempotency (reject duplicates gracefully)
- Enforce `campus_id` from the JWT (not from the payload)
- Create audit log entries for each synced record
- Return per-record status: `created`, `conflict`, or `error`

If the backend does not yet handle the entity type in sync push, implement it before proceeding.

### Step 6: Verify Sync Pull Endpoint

Confirm that `GET /api/v1/sync/pull?since=<timestamp>&entity_types=<type>` returns the entity type.

The sync pull handler fetches all records updated since the given timestamp, scoped to the user's campus_id. The frontend uses this to update local Dexie data after reconnecting.

```typescript
// frontend/src/offline/sync/syncPull.ts

import { db } from '../db';
import { syncApi } from '../../api/syncApi';

export async function pullUpdates(since: string): Promise<void> {
  const response = await syncApi.pull({ since, entity_types: 'person,triage,attendance' });

  for (const record of response.records) {
    switch (record.entity_type) {
      case 'person':
        await db.persons.put({
          id: record.id,
          syncId: record.id,
          data: record.data,
          syncStatus: 'synced',
          localCreatedAt: record.data.created_at,
          serverUpdatedAt: record.updated_at,
        });
        break;
      // Handle other entity types...
    }
  }

  // Store the server timestamp for the next pull
  localStorage.setItem('lastSyncTimestamp', response.server_timestamp);
}
```

### Step 7: Show Sync Status Indicator

Display the sync status to the user so they know whether their data has been uploaded:

```typescript
// frontend/src/components/common/SyncStatusBadge.tsx

interface SyncStatusBadgeProps {
  status: 'pending' | 'synced' | 'conflict';
}

export function SyncStatusBadge({ status }: SyncStatusBadgeProps) {
  const config = {
    pending: { label: 'Pendente', className: 'bg-yellow-100 text-yellow-800' },
    synced: { label: 'Sincronizado', className: 'bg-green-100 text-green-800' },
    conflict: { label: 'Conflito', className: 'bg-red-100 text-red-800' },
  };

  const { label, className } = config[status];

  return (
    <span className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${className}`}>
      {label}
    </span>
  );
}
```

Also show a global sync indicator in the app header:

```typescript
// frontend/src/components/common/GlobalSyncStatus.tsx

import { useLiveQuery } from 'dexie-react-hooks';
import { db } from '../../offline/db';

export function GlobalSyncStatus() {
  const pendingCount = useLiveQuery(
    () => db.syncQueue.where('syncStatus').equals('pending').count()
  );

  if (!pendingCount) return null;

  return (
    <div className="flex items-center gap-1 text-xs text-yellow-700 bg-yellow-50 px-2 py-1 rounded">
      <span className="inline-block h-2 w-2 rounded-full bg-yellow-500 animate-pulse" />
      {pendingCount} {pendingCount === 1 ? 'alteracao pendente' : 'alteracoes pendentes'}
    </div>
  );
}
```

For **online-only** features, show a clear message when offline:

```typescript
export function OnlineOnlyGuard({ children }: { children: React.ReactNode }) {
  const isOnline = useOnlineStatus();

  if (!isOnline) {
    return (
      <div className="flex items-center justify-center p-8 text-gray-500">
        <p>Esta funcionalidade requer conexao com a internet.</p>
      </div>
    );
  }

  return <>{children}</>;
}
```

### Step 8: Encrypt Sensitive Fields

Reference: `docs/12-offline-sync-strategy.md`, `docs/18-threat-model.md` (T4: Offline Device Theft)

Sensitive fields stored in IndexedDB must be encrypted using the Web Crypto API with AES-256-GCM:

```typescript
// frontend/src/offline/encryption.ts

const ALGORITHM = 'AES-GCM';
const KEY_LENGTH = 256;

async function getEncryptionKey(): Promise<CryptoKey> {
  // Derive key from user's Keycloak token or a stored key
  // The key is created on first login and stored in a secure manner
  const storedKey = localStorage.getItem('encryptionKeyJwk');
  if (storedKey) {
    return crypto.subtle.importKey(
      'jwk',
      JSON.parse(storedKey),
      { name: ALGORITHM, length: KEY_LENGTH },
      false,
      ['encrypt', 'decrypt']
    );
  }

  // Generate new key
  const key = await crypto.subtle.generateKey(
    { name: ALGORITHM, length: KEY_LENGTH },
    true,
    ['encrypt', 'decrypt']
  );

  const exported = await crypto.subtle.exportKey('jwk', key);
  localStorage.setItem('encryptionKeyJwk', JSON.stringify(exported));
  return key;
}

export async function encryptSensitiveFields(
  fields: Record<string, string | undefined>
): Promise<string> {
  const key = await getEncryptionKey();
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encoder = new TextEncoder();
  const data = encoder.encode(JSON.stringify(fields));

  const encrypted = await crypto.subtle.encrypt(
    { name: ALGORITHM, iv },
    key,
    data
  );

  // Store IV + ciphertext together
  return JSON.stringify({
    iv: Array.from(iv),
    data: Array.from(new Uint8Array(encrypted)),
  });
}

export async function decryptSensitiveFields(
  encryptedBlob: string
): Promise<Record<string, string | undefined>> {
  const key = await getEncryptionKey();
  const { iv, data } = JSON.parse(encryptedBlob);

  const decrypted = await crypto.subtle.decrypt(
    { name: ALGORITHM, iv: new Uint8Array(iv) },
    key,
    new Uint8Array(data)
  );

  const decoder = new TextDecoder();
  return JSON.parse(decoder.decode(decrypted));
}
```

Fields that must be encrypted in IndexedDB:
- `document_number` (CPF, SSN, etc.)
- Health-related data from `assisted_profile`
- Any field classified as "High" or "Critical" in `docs/18-threat-model.md` asset table

Fields that are safe to store unencrypted:
- `full_name` (needed for local search)
- `id`, `sync_id`, `campus_id`
- `syncStatus`, timestamps

### Step 9: Test Offline Scenarios

Manual test procedure:

1. **Disconnect test**: Open browser DevTools > Network tab > Set to "Offline"
2. **Create while offline**: Fill and submit the form. Verify:
   - Record appears in the local list
   - Sync status shows "Pendente"
   - No network errors shown to user
   - Record saved in IndexedDB (DevTools > Application > IndexedDB > chesed)
3. **Reconnect and sync**: Set network back to "Online". Verify:
   - Sync queue is processed automatically
   - Sync status changes to "Sincronizado"
   - Record exists on the server (check via API or database)
   - Server-generated fields (created_at, server id) are updated locally
4. **Conflict test**: Create a record offline, then create the same sync_id on the server directly. Reconnect and verify:
   - Sync reports "conflict" status
   - User can see conflict indicator
5. **Data persistence test**: Create records offline, close the browser, reopen. Verify records still in IndexedDB.
6. **Encryption test**: Create a record with PII offline. Check IndexedDB directly (DevTools). Verify `document_number` is not visible in plaintext.

Automated test (Vitest):

```typescript
import { createPersonOffline } from '../offline/operations/personOperations';
import { db } from '../offline/db';

describe('Offline person creation', () => {
  beforeEach(async () => {
    await db.persons.clear();
    await db.syncQueue.clear();
  });

  it('stores person locally with pending sync status', async () => {
    const id = await createPersonOffline({
      full_name: 'Test Person',
      document_type: 'CPF',
    });

    const stored = await db.persons.get(id);
    expect(stored).toBeDefined();
    expect(stored!.syncStatus).toBe('pending');

    const queueItems = await db.syncQueue
      .where('syncId').equals(id)
      .toArray();
    expect(queueItems).toHaveLength(1);
    expect(queueItems[0].entityType).toBe('person');
  });
});
```

---

## Checklist

- [ ] Feature classified (fully offline / read-only offline / online-only)
- [ ] Dexie table added with appropriate indexes
- [ ] Client-side UUID generation uses `crypto.randomUUID()`
- [ ] Sync queue entry created on local save
- [ ] `POST /api/v1/sync/push` handles the entity type
- [ ] `GET /api/v1/sync/pull` returns the entity type
- [ ] Sync status badge displayed (pending/synced/conflict)
- [ ] Global sync indicator shows pending count
- [ ] Sensitive fields encrypted with AES-256-GCM before IndexedDB storage
- [ ] Online-only features show appropriate message when offline
- [ ] Disconnect-create-reconnect tested manually
- [ ] Conflict scenario tested
- [ ] Automated tests for offline operations
