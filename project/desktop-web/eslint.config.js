// ESLint 9 配置文件 (Flat Config)
import js from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
    // 忽略的文件
    {
        ignores: [
            'dist/**',
            'node_modules/**',
            'src/protocol/**',  // FlatBuffers 生成的代码
            '**/*.cjs',
            'vite.config.ts'
        ],
    },

    // JavaScript 推荐规则
    js.configs.recommended,

    // TypeScript 推荐规则
    ...tseslint.configs.recommended,

    // 自定义规则
    {
        rules: {
            // TypeScript 规则
            '@typescript-eslint/no-explicit-any': 'warn',
            '@typescript-eslint/no-unused-vars': ['warn', {
                argsIgnorePattern: '^_',
                varsIgnorePattern: '^_',
            }],

            // 通用规则
            'no-console': ['warn', { allow: ['warn', 'error'] }],
            'no-debugger': 'warn',
        },
    }
);
