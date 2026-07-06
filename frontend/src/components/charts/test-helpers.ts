/**
 * Recharts' ResponsiveContainer observes its parent via ResizeObserver, which
 * jsdom does not implement. Chart tests import this module for its side effect
 * to install a no-op polyfill so the container mounts instead of throwing.
 */
class ResizeObserverStub implements ResizeObserver {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}

if (!('ResizeObserver' in globalThis)) {
  globalThis.ResizeObserver = ResizeObserverStub;
}

export {};
