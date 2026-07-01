import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest';
import 'fake-indexeddb/auto';
import {
  encryptPayload,
  decryptPayload,
  isEncryptionAvailable,
  resetEncryptionKeyCache,
  type StoredPayload,
} from '../encryption';
import { db } from '../db';

// These tests exercise both encryption branches required by S05.1 criterion 3:
// the supported path (real AES-GCM via Web Crypto) and the graceful-degradation
// path (crypto.subtle absent → plaintext passthrough, app still functions).
describe('offline encryption', () => {
  beforeEach(async () => {
    await db.delete();
    await db.open();
    resetEncryptionKeyCache();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    resetEncryptionKeyCache();
  });

  describe('when Web Crypto is available (supported browser)', () => {
    it('reports encryption as available', () => {
      expect(isEncryptionAvailable()).toBe(true);
    });

    it('round-trips an object through encrypt/decrypt', async () => {
      const original = { full_name: 'Maria Santos', document_number: '123' };
      const stored = await encryptPayload(original);
      const restored = await decryptPayload(stored);
      expect(restored).toEqual(original);
    });

    it('stores ciphertext, not plaintext', async () => {
      const original = { secret: 'sensitive-value' };
      const stored = await encryptPayload(original);
      expect(stored.enc).toBe(true);
      const serialized = JSON.stringify(stored);
      expect(serialized).not.toContain('sensitive-value');
    });

    it('uses a fresh IV per encryption so identical inputs differ', async () => {
      const a = (await encryptPayload({ x: 1 })) as Extract<
        StoredPayload,
        { enc: true }
      >;
      const b = (await encryptPayload({ x: 1 })) as Extract<
        StoredPayload,
        { enc: true }
      >;
      expect(a.iv).not.toEqual(b.iv);
    });

    it('reuses a persisted key across a cache reset (reload durability)', async () => {
      const stored = await encryptPayload({ v: 'durable' });
      // Simulate a page reload: drop the in-memory key, force re-read from store.
      resetEncryptionKeyCache();
      const restored = await decryptPayload(stored);
      expect(restored).toEqual({ v: 'durable' });
    });
  });

  describe('when Web Crypto is unavailable (graceful degradation)', () => {
    beforeEach(() => {
      // Emulate a browser without SubtleCrypto (e.g. non-secure context).
      vi.stubGlobal('crypto', {
        getRandomValues: (a: Uint8Array) => a,
      });
      resetEncryptionKeyCache();
    });

    it('reports encryption as unavailable', () => {
      expect(isEncryptionAvailable()).toBe(false);
    });

    it('stores plaintext and still round-trips', async () => {
      const original = { full_name: 'Fallback User' };
      const stored = await encryptPayload(original);
      expect(stored.enc).toBe(false);
      const restored = await decryptPayload(stored);
      expect(restored).toEqual(original);
    });
  });

  it('decrypts a plaintext record even while encryption is available (backward compat)', async () => {
    const legacy: StoredPayload = { enc: false, data: { legacy: true } };
    const restored = await decryptPayload(legacy);
    expect(restored).toEqual({ legacy: true });
  });
});
