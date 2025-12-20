import { create } from 'zustand';
import { apiClient } from '@/services/api';

// 好友信息
export interface Friend {
    id: string;           // 好友关系 ID
    friend_id: string;    // 好友用户 ID
    username: string;
    nickname: string;
    avatar: string;
    remark: string;
    create_at: string;
}

// 好友请求
export interface FriendRequest {
    id: string;
    from_user_id: string;
    from_username: string;
    from_nickname: string;
    from_avatar: string;
    message: string;
    status: number;
    create_at: string;
}

interface FriendState {
    friends: Friend[];
    friendRequests: FriendRequest[];
    loading: boolean;
    error: string | null;

    // Actions
    fetchFriends: () => Promise<void>;
    fetchFriendRequests: () => Promise<void>;
    sendFriendRequest: (friendId: string, message?: string) => Promise<boolean>;
    acceptRequest: (requestId: string) => Promise<boolean>;
    rejectRequest: (requestId: string) => Promise<boolean>;
    deleteFriend: (friendId: string) => Promise<boolean>;
    clearError: () => void;
}

export const useFriendStore = create<FriendState>((set, get) => ({
    friends: [],
    friendRequests: [],
    loading: false,
    error: null,

    // 获取好友列表
    fetchFriends: async () => {
        set({ loading: true, error: null });
        try {
            const response = await apiClient.get<{ list: Friend[] }>('/friends');
            if (response.data.code === 0) {
                set({ friends: response.data.data?.list || [], loading: false });
            } else {
                set({ error: response.data.message, loading: false });
            }
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '获取好友列表失败';
            set({ error: message, loading: false });
        }
    },

    // 获取好友请求列表
    fetchFriendRequests: async () => {
        set({ loading: true, error: null });
        try {
            const response = await apiClient.get<{ list: FriendRequest[] }>('/friends/requests');
            if (response.data.code === 0) {
                set({ friendRequests: response.data.data?.list || [], loading: false });
            } else {
                set({ error: response.data.message, loading: false });
            }
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '获取好友请求失败';
            set({ error: message, loading: false });
        }
    },

    // 发送好友请求
    sendFriendRequest: async (friendId: string, message?: string) => {
        try {
            const response = await apiClient.post('/friends/request', {
                friend_id: parseInt(friendId),
                message: message || '',
            });
            if (response.data.code === 0) {
                return true;
            }
            set({ error: response.data.message });
            return false;
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '发送好友请求失败';
            set({ error: message });
            return false;
        }
    },

    // 接受好友请求
    acceptRequest: async (requestId: string) => {
        try {
            const response = await apiClient.post(`/friends/accept/${requestId}`);
            if (response.data.code === 0) {
                // 刷新好友列表和请求列表
                await get().fetchFriends();
                await get().fetchFriendRequests();
                return true;
            }
            set({ error: response.data.message });
            return false;
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '接受好友请求失败';
            set({ error: message });
            return false;
        }
    },

    // 拒绝好友请求
    rejectRequest: async (requestId: string) => {
        try {
            const response = await apiClient.post(`/friends/reject/${requestId}`);
            if (response.data.code === 0) {
                await get().fetchFriendRequests();
                return true;
            }
            set({ error: response.data.message });
            return false;
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '拒绝好友请求失败';
            set({ error: message });
            return false;
        }
    },

    // 删除好友
    deleteFriend: async (friendId: string) => {
        try {
            const response = await apiClient.delete(`/friends/${friendId}`);
            if (response.data.code === 0) {
                await get().fetchFriends();
                return true;
            }
            set({ error: response.data.message });
            return false;
        } catch (error: unknown) {
            const message = error instanceof Error ? error.message : '删除好友失败';
            set({ error: message });
            return false;
        }
    },

    clearError: () => set({ error: null }),
}));
