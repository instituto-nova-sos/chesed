import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';
import eslintConfigPrettier from 'eslint-config-prettier';
import { defineConfig, globalIgnores } from 'eslint/config';

export default defineConfig([
  globalIgnores(['dist', 'coverage', 'dev-dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
      // Downgraded to 'warn': false positives on async fetch-then-setState
      // patterns. Sprint 3 introduces TanStack Query to eliminate the pattern,
      // after which this can be re-promoted to 'error'.
      'react-hooks/set-state-in-effect': 'warn',
      // CLAUDE.md complexity thresholds. Sprint 2.5 sets these as warnings;
      // Sprint 3/4 refactors violations and ratchets to 'error'.
      complexity: ['warn', 10],
      'max-depth': ['warn', 3],
      'max-params': ['warn', 5],
      'max-lines-per-function': [
        'warn',
        { max: 80, skipBlankLines: true, skipComments: true },
      ],
      'max-lines': [
        'warn',
        { max: 300, skipBlankLines: true, skipComments: true },
      ],
    },
  },
  // Playwright E2E suite runs in Node, not the browser, and the Playwright
  // fixture API relies on empty destructuring patterns (`async ({}, use) =>`).
  {
    files: ['e2e/**/*.ts'],
    languageOptions: {
      globals: globals.node,
    },
    rules: {
      'no-empty-pattern': 'off',
      'react-hooks/rules-of-hooks': 'off',
    },
  },
  // Test files (Vitest unit/integration). The function-length and complexity
  // budgets target production maintainability; test suites legitimately have long
  // describe/it blocks and table-driven cases whose arrange-act-assert flow is
  // clearer kept together than fragmented to satisfy a line count. This mirrors
  // the coverage config, which already excludes tests from the production budget.
  {
    files: ['**/*.{test,spec}.{ts,tsx}', '**/__tests__/**/*.{ts,tsx}'],
    rules: {
      'max-lines-per-function': 'off',
      'max-lines': 'off',
      complexity: 'off',
    },
  },
  eslintConfigPrettier,
]);
