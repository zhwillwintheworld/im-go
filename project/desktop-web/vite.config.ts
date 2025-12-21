import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
        },
    },
    server: {
        host: '0.0.0.0',  // 允许局域网访问
        port: 8083,
        proxy: {
            '/api': {
                target: 'http://localhost:8082',
                changeOrigin: true,
            }
        },
    },
    build: {
        target: 'esnext',
        minify: 'esbuild',
        sourcemap: true,
        rollupOptions: {
            output: {
                manualChunks: {
                    vendor: ['react', 'react-dom', 'react-router-dom'],
                    antd: ['antd', '@ant-design/icons'],
                    utils: ['dayjs', 'flatbuffers'],
                },
            },
        },
    },
});
