// security-base 共通 ESLint 設定 (flat config / ESLint 9+)
//
// 利用側の eslint.config.js で読み込み、言語固有の設定 (typescript-eslint 等) と組み合わせる:
//
//   import securityBase from './configs/eslint.config.base.mjs';
//   import tseslint from 'typescript-eslint';
//   export default [...securityBase, ...tseslint.configs.recommended];
//
// 必要な devDependencies: eslint, @eslint/js, eslint-plugin-security, globals
import js from '@eslint/js';
import security from 'eslint-plugin-security';
import globals from 'globals';

export default [
  {
    ignores: ['dist/', 'build/', 'coverage/', 'node_modules/'],
  },
  js.configs.recommended,
  security.configs.recommended,
  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.node,
      },
    },
    rules: {
      // 誤検知が多いものは warn に留める
      'security/detect-object-injection': 'warn',
      'security/detect-non-literal-fs-filename': 'warn',
      'security/detect-child-process': 'warn',
      'security/detect-possible-timing-attacks': 'warn',
      // 実害に直結しやすいものは error
      'security/detect-unsafe-regex': 'error',
      'security/detect-non-literal-regexp': 'error',
      'security/detect-non-literal-require': 'error',
      'security/detect-eval-with-expression': 'error',
      'security/detect-buffer-noassert': 'error',
      'security/detect-new-buffer': 'error',
      'security/detect-pseudoRandomBytes': 'error',
      'security/detect-disable-mustache-escape': 'error',
      'security/detect-no-csrf-before-method-override': 'error',
      'security/detect-bidi-characters': 'error',
      'security/detect-invisible-characters': 'error',
    },
  },
];
