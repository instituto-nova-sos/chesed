import { describe, expect, it } from 'vitest';
import Dexie from 'dexie';
import { db } from '../db';

/**
 * These tests assert the Dexie schema declaration only. They deliberately do
 * NOT open the database: jsdom ships no IndexedDB and the project has no
 * fake-indexeddb polyfill, so `db.open()` / table reads are exercised in the
 * MSW-backed integration suite and (later) the sync drainer suite instead.
 * The schema metadata below is populated synchronously by `version().stores()`
 * in the ChesedDB constructor, so it is observable without an open connection.
 */
describe('offline db schema', () => {
  it('is a Dexie database named "chesed-offline"', () => {
    expect(db).toBeInstanceOf(Dexie);
    expect(db.name).toBe('chesed-offline');
  });

  it('declares schema version 3', () => {
    expect(db.verno).toBe(3);
  });

  it('exposes the persons, syncQueue, syncMeta, triages, attendances and conflicts tables', () => {
    const tableNames = db.tables.map((t) => t.name).sort();
    expect(tableNames).toEqual(
      ['persons', 'syncQueue', 'syncMeta', 'triages', 'attendances', 'conflicts'].sort(),
    );
  });

  it('keys persons on "id" and indexes syncStatus', () => {
    const persons = db.table('persons');
    expect(persons.schema.primKey.name).toBe('id');
    expect(persons.schema.primKey.auto).toBe(false);
    const indexNames = persons.schema.indexes.map((idx) => idx.name);
    expect(indexNames).toContain('syncStatus');
  });

  it('uses an auto-incrementing primary key for syncQueue', () => {
    const syncQueue = db.table('syncQueue');
    expect(syncQueue.schema.primKey.name).toBe('id');
    expect(syncQueue.schema.primKey.auto).toBe(true);
    const indexNames = syncQueue.schema.indexes.map((idx) => idx.name);
    expect(indexNames).toEqual(
      expect.arrayContaining(['entityType', 'entityId', 'createdAt']),
    );
  });

  it('keys syncMeta on "key"', () => {
    const syncMeta = db.table('syncMeta');
    expect(syncMeta.schema.primKey.name).toBe('key');
  });
});
