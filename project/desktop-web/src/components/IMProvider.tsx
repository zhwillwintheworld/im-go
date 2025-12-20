import { useEffect, useState, ReactNode } from 'react';
import { useAuthStore } from '@/stores/authStore';
import { useIMStore } from '@/stores/imStore';
import { messageDispatcher } from '@/services/messageDispatcher';

interface IMProviderProps {
    children: ReactNode;
}

/**
 * IM 连接管理 Provider
 * 监听登录状态，自动管理 IM 连接生命周期
 */
export function IMProvider({ children }: IMProviderProps) {
    const [initialized, setInitialized] = useState(false);
    const token = useAuthStore((state) => state.token);
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
    const checkAuth = useAuthStore((state) => state.checkAuth);
    const imConnect = useIMStore((state) => state.connect);
    const imDisconnect = useIMStore((state) => state.disconnect);

    // 初始化：恢复登录状态
    useEffect(() => {
        checkAuth();
        messageDispatcher.init();
        setInitialized(true);
    }, [checkAuth]);

    // 监听认证状态变化，管理 IM 连接
    useEffect(() => {
        // 等待初始化完成再开始监听
        if (!initialized) return;

        if (isAuthenticated && token) {
            console.log('[IMProvider] User authenticated, connecting to IM...');
            imConnect(token).catch((err) => {
                console.error('[IMProvider] IM connect failed:', err);
            });
        } else {
            console.log('[IMProvider] User not authenticated, disconnecting IM...');
            imDisconnect();
        }
    }, [initialized, isAuthenticated, token, imConnect, imDisconnect]);

    return <>{children}</>;
}
