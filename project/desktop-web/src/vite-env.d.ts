/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_ENV: 'development' | 'test' | 'production';
    readonly VITE_API_BASE_URL: string;
    readonly VITE_ACCESS_URL: string;
    readonly VITE_LOG_LEVEL: 'debug' | 'info' | 'warn' | 'error';
}

interface ImportMeta {
    readonly env: ImportMetaEnv;
}
