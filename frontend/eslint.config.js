import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';
import eslintConfigPrettier from 'eslint-config-prettier';
import { defineConfig, globalIgnores } from 'eslint/config';

export default defineConfig([
  globalIgnores(['dist']),
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
  eslintConfigPrettier,
]);
