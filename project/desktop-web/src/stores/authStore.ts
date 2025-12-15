import { create } from 'zustand';

interface User {
    id: string;
    username: string;
    nickname: string;
    avatar: string;
}

interface AuthState {
    user: User | null;
    token: string | null;
    isAuthenticated: boolean;
    login: (username: string, password: string) => Promise<void>;
    logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
    user: null,
    token: null,
    isAuthenticated: false,

    login: async (username: string, password: string) => {
        // TODO: 调用登录 API
        const mockUser: User = {
            id: '1',
            username,
            nickname: username,
            avatar: '',
        };
        const mockToken = 'mock-token';

        set({
            user: mockUser,
            token: mockToken,
            isAuthenticated: true,
        });

        // 存储到 localStorage
        localStorage.setItem('token', mockToken);
    },

    logout: () => {
        set({
            user: null,
            token: null,
            isAuthenticated: false,
        });
        localStorage.removeItem('token');
    },
}));
