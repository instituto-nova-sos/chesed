import { db } from './db';

/**
 * StoredPayload is the self-describing on-disk shape for an offline record's
 * `data` field. The `enc` discriminant lets the read path decrypt correctly
 * regardless of which branch wrote the record — so plaintext records written
 * before encryption was available (or on an unsupported browser) remain
 * readable, and vice versa.
 */
export type StoredPayload =
  | { enc: false; data: Record<string, unknown> }
  | { enc: true; iv: number[]; ciphertext: number[] };

const KEY_META_ID = 'enc:key';
const IV_BYTES = 12; // AES-GCM standard nonce length.

let cachedKey: CryptoKey | null = null;

/**
 * isEncryptionAvailable feature-detects the Web Crypto primitives required for
 * at-rest encryption. When false, the offline layer degrades gracefully to
 * plaintext storage and the app keeps functioning (S05.1 criterion 3).
 */
export function isEncryptionAvailable(): boolean {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle || typeof globalThis.crypto?.getRandomValues !== 'function') {
    return false;
  }
  const required = ['encrypt', 'decrypt', 'generateKey', 'exportKey', 'importKey'] as const;
  return required.every((op) => typeof subtle[op] === 'function');
}

/**
 * resetEncryptionKeyCache drops the in-memory key so the next operation re-reads
 * it from the durable store. Used to simulate a page reload in tests.
 */
export function resetEncryptionKeyCache(): void {
  cachedKey = null;
}

function toBase64(bytes: Uint8Array): string {
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary);
}

// Web Crypto expects an ArrayBuffer-backed view. Copying the bytes into a view
// over a freshly-allocated ArrayBuffer guarantees the concrete `ArrayBuffer`
// type the crypto overloads require (not the `ArrayBufferLike` union, which may
// widen to SharedArrayBuffer under the build's lib settings) and is accepted by
// the jsdom SubtleCrypto implementation as a TypedArray.
function toBufferView(bytes: Uint8Array): Uint8Array<ArrayBuffer> {
  const copy = new Uint8Array(new ArrayBuffer(bytes.byteLength));
  copy.set(bytes);
  return copy;
}

function fromBase64(value: string): Uint8Array<ArrayBuffer> {
  const binary = atob(value);
  const bytes = new Uint8Array(new ArrayBuffer(binary.length));
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

async function loadOrCreateKey(): Promise<CryptoKey> {
  if (cachedKey) return cachedKey;

  const existing = await db.syncMeta.get(KEY_META_ID);
  if (existing) {
    cachedKey = await crypto.subtle.importKey(
      'raw',
      fromBase64(existing.value),
      { name: 'AES-GCM' },
      false,
      ['encrypt', 'decrypt'],
    );
    return cachedKey;
  }

  const key = await crypto.subtle.generateKey(
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt'],
  );
  const raw = new Uint8Array(await crypto.subtle.exportKey('raw', key));
  await db.syncMeta.put({ key: KEY_META_ID, value: toBase64(raw) });
  cachedKey = key;
  return cachedKey;
}

/**
 * encryptPayload encrypts a record's data with AES-GCM when Web Crypto is
 * available, and otherwise returns it as plaintext so the app still works.
 */
export async function encryptPayload(
  data: Record<string, unknown>,
): Promise<StoredPayload> {
  if (!isEncryptionAvailable()) {
    return { enc: false, data };
  }

  const key = await loadOrCreateKey();
  const iv = crypto.getRandomValues(new Uint8Array(IV_BYTES));
  const encoded = new TextEncoder().encode(JSON.stringify(data));
  const buffer = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: toBufferView(iv) },
    key,
    toBufferView(encoded),
  );

  return {
    enc: true,
    iv: Array.from(iv),
    ciphertext: Array.from(new Uint8Array(buffer)),
  };
}

/**
 * decryptPayload restores a record's data. It branches on the stored `enc`
 * discriminant, so plaintext records are returned as-is (backward compatible)
 * and encrypted records are decrypted with the persisted key.
 */
export async function decryptPayload(
  stored: StoredPayload,
): Promise<Record<string, unknown>> {
  if (!stored.enc) {
    return stored.data;
  }

  const key = await loadOrCreateKey();
  const iv = toBufferView(new Uint8Array(stored.iv));
  const ciphertext = toBufferView(new Uint8Array(stored.ciphertext));
  const buffer = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv },
    key,
    ciphertext,
  );
  return JSON.parse(new TextDecoder().decode(buffer)) as Record<string, unknown>;
}
