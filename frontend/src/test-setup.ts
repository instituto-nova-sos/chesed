import '@testing-library/jest-dom';
// Provide an in-memory IndexedDB to every test. jsdom ships none, and any test
// that mounts a component reaching the offline layer (useOnlineSync → Dexie)
// would otherwise throw DatabaseClosedError. Tests needing isolation still
// reset the database explicitly in their own beforeEach.
import 'fake-indexeddb/auto';
