import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useOfflineStatus } from '../useOfflineStatus';

function setNavigatorOnline(value: boolean): void {
  Object.defineProperty(navigator, 'onLine', {
    configurable: true,
    value,
  });
}

describe('useOfflineStatus', () => {
  afterEach(() => {
    setNavigatorOnline(true);
    vi.restoreAllMocks();
  });

  it('reports offline based on navigator.onLine at mount', () => {
    setNavigatorOnline(false);
    const { result } = renderHook(() => useOfflineStatus());
    expect(result.current.isOffline).toBe(true);
  });

  it('reports online when navigator.onLine is true at mount', () => {
    setNavigatorOnline(true);
    const { result } = renderHook(() => useOfflineStatus());
    expect(result.current.isOffline).toBe(false);
  });

  it('flips to offline when the window emits an offline event', () => {
    setNavigatorOnline(true);
    const { result } = renderHook(() => useOfflineStatus());
    expect(result.current.isOffline).toBe(false);

    act(() => {
      window.dispatchEvent(new Event('offline'));
    });

    expect(result.current.isOffline).toBe(true);
  });

  it('flips back to online when the window emits an online event', () => {
    setNavigatorOnline(false);
    const { result } = renderHook(() => useOfflineStatus());
    expect(result.current.isOffline).toBe(true);

    act(() => {
      window.dispatchEvent(new Event('online'));
    });

    expect(result.current.isOffline).toBe(false);
  });

  it('removes its window listeners on unmount', () => {
    const removeSpy = vi.spyOn(window, 'removeEventListener');
    const { unmount } = renderHook(() => useOfflineStatus());

    unmount();

    const removedEvents = removeSpy.mock.calls.map(([event]) => event);
    expect(removedEvents).toContain('online');
    expect(removedEvents).toContain('offline');
  });
});
