import { create } from 'zustand';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol } from '@/services/protocol/IMProtocol';
import { config } from '@/config';

type IMConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'authenticating' | 'authenticated' | 'error';

interface IMState {
    status: IMConnectionStatus;
    error: string | null;
    connect: (token: string) => Promise<void>;
    disconnect: () => void;
}

// 获取设备 ID
const getDeviceId = (): string => {
    let deviceId = localStorage.getItem('device_id');
    if (!deviceId) {
        deviceId = crypto.randomUUID();
        localStorage.setItem('device_id', deviceId);
    }
    return deviceId;
};

export const useIMStore = create<IMState>((set, get) => ({
    status: 'disconnected',
    error: null,

    connect: async (token: string) => {
        const currentStatus = get().status;
        if (currentStatus === 'connected' || currentStatus === 'authenticated' ||
            currentStatus === 'connecting' || currentStatus === 'authenticating') {
            return;
        }

        try {
            set({ status: 'connecting', error: null });

            // 建立 WebTransport 连接并自动发送认证请求
            await transportManager.connect(config.webTransportUrl, {
                token,
                deviceId: getDeviceId(),
                appVersion: '1.0.0'
            });

            // 认证成功
            set({ status: 'authenticated' });
            console.log('[IMStore] Connected and authenticated');
        } catch (err) {
            const errorMsg = err instanceof Error ? err.message : 'Unknown error';
            console.error('[IMStore] Connection failed:', errorMsg);
            set({ status: 'error', error: errorMsg });
            throw err;
        }
    },

    disconnect: () => {
        transportManager.disconnect();
        set({ status: 'disconnected', error: null });
        console.log('[IMStore] Disconnected');
    },
}));

// 监听 WebTransport 状态变化
transportManager.onStatusChange((status) => {
    if (status === 'disconnected') {
        useIMStore.setState({ status: 'disconnected' });
    } else if (status === 'reconnecting') {
        useIMStore.setState({ status: 'connecting' });
    }
});
