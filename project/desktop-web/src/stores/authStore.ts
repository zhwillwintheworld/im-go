import { create } from 'zustand';
import { config } from '@/config';

interface User {
    id: string;
    username: string;
    nickname: string;
    avatar: string;
}

interface AuthState {
    user: User | null;
    token: string | null;
    refreshToken: string | null;
    isAuthenticated: boolean;
    login: (username: string, password: string) => Promise<void>;
    register: (username: string, password: string, nickname: string) => Promise<void>;
    logout: () => Promise<void>;
    checkAuth: () => void;
}

interface ApiResponse<T> {
    code: number;
    message: string;
    data: T;
}

interface LoginResponse {
    accessToken: string;
    refreshToken: string;
    userId: string;  // 使用 string 防止大整数精度丢失
    expiresAt: number;
}

// 生成设备ID
const getDeviceId = (): string => {
    let deviceId = localStorage.getItem('device_id');
    if (!deviceId) {
        deviceId = crypto.randomUUID();
        localStorage.setItem('device_id', deviceId);
    }
    return deviceId;
};

export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    token: null,
    refreshToken: null,
    isAuthenticated: false,

    login: async (username: string, password: string) => {
        const response = await fetch(`${config.apiBaseUrl}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                username,
                password,
                deviceId: getDeviceId(),
                platform: 'web',
            }),
        });

        const result: ApiResponse<LoginResponse> = await response.json();

        if (result.code !== 0) {
            throw new Error(result.message || '登录失败');
        }

        const { accessToken, refreshToken, userId } = result.data;

        // 保存 token
        localStorage.setItem('token', accessToken);
        localStorage.setItem('refresh_token', refreshToken);

        set({
            token: accessToken,
            refreshToken: refreshToken,
            isAuthenticated: true,
        });

        // 获取用户信息
        const profileResponse = await fetch(`${config.apiBaseUrl}/api/v1/user/profile`, {
            headers: {
                'Authorization': accessToken,
            },
        });

        const profileResult: ApiResponse<User> = await profileResponse.json();

        if (profileResult.code === 0) {
            set({ user: profileResult.data });
        } else {
            // 如果获取用户信息失败，使用基本信息
            set({
                user: {
                    id: String(userId),
                    username,
                    nickname: username,
                    avatar: '',
                },
            });
        }
    },

    register: async (username: string, password: string, nickname: string) => {
        const response = await fetch(`${config.apiBaseUrl}/api/v1/auth/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                username,
                password,
                nickname,
            }),
        });

        const result: ApiResponse<null> = await response.json();

        if (result.code !== 0) {
            throw new Error(result.message || '注册失败');
        }
    },

    logout: async () => {
        const token = get().token;

        if (token) {
            try {
                await fetch(`${config.apiBaseUrl}/api/v1/auth/logout`, {
                    method: 'POST',
                    headers: {
                        'Authorization': token,
                    },
                });
            } catch (e) {
                console.error('Logout API error:', e);
            }
        }

        set({
            user: null,
            token: null,
            refreshToken: null,
            isAuthenticated: false,
        });
        localStorage.removeItem('token');
        localStorage.removeItem('refresh_token');
    },

    checkAuth: () => {
        const token = localStorage.getItem('token');
        const refreshToken = localStorage.getItem('refresh_token');

        if (token) {
            set({
                token,
                refreshToken,
                isAuthenticated: true,
            });
        }
    },
}));
