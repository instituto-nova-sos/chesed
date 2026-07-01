/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { VitePWA } from 'vite-plugin-pwa';
import { resolve } from 'path';

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'Chesed - Instituto Nova SOS',
        short_name: 'Chesed',
        description: 'Social services management platform',
        theme_color: '#ffffff',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        icons: [],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test-setup.ts',
    // Vitest owns the unit/integration suite under src/. Playwright owns e2e/
    // (those specs import @playwright/test and must never be collected here).
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['node_modules/**', 'dist/**', 'e2e/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'json-summary'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.{test,spec}.{ts,tsx}',
        'src/test-setup.ts',
        'src/main.tsx',
        'src/vite-env.d.ts',
        'src/types/**',
      ],
      // Coverage floor for the offline-sync baseline (tasks D1/D2).
      //
      // Enforced today (all green): the offline persistence layer and the two
      // sync-drainer dependencies (offline-status hook, auth store) are held at
      // 80% so the upcoming drainer starts from a tested foundation. Per-glob
      // thresholds are checked against ONLY the files matching each glob, so a
      // single untested sibling cannot mask a regression in these modules.
      //
      // Targets to ratchet toward as the suite grows (NOT yet enforced because
      // the broader app sits at ~4.5% overall, well below these floors — raising
      // the global gate now would break `test:coverage` rather than guard it):
      //   - Global: 50% (branches/functions/lines/statements)
      //   - src/hooks/**: 80%  (only useOfflineStatus is covered so far)
      // Bump the bare global keys to 50 and broaden `src/hooks/**` to the whole
      // folder once those directories clear the bar; never lower an enforced
      // value — only ratchet up.
      thresholds: {
        branches: 0,
        functions: 0,
        lines: 0,
        statements: 0,
        'src/offline/**': {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
        'src/hooks/useOfflineStatus.ts': {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
        'src/store/authStore.ts': {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
      },
    },
  },
});
