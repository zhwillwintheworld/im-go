import { create } from 'zustand';
import { Friend } from './friendStore';

export interface Conversation {
    id: string;           // 对话 ID（使用对方用户 ID）
    name: string;         // 显示名称
    avatar: string;       // 头像
    lastMessage: string;  // 最后一条消息
    unreadCount: number;  // 未读数
    updatedAt: number;    // 最后更新时间
}

interface ChatState {
    conversations: Conversation[];
    activeConversationId: string | null;

    // Actions
    setActiveConversation: (id: string | null) => void;
    updateConversation: (conv: Conversation) => void;
    addConversationFromFriend: (friend: Friend) => void;
    removeConversation: (convId: string) => void;
    markAsRead: (convId: string) => void;
    updateLastMessage: (convId: string, message: string) => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
    conversations: [],
    activeConversationId: null,

    setActiveConversation: (id: string | null) => {
        set({ activeConversationId: id });
        // 设置当前对话时自动标记已读
        if (id) {
            get().markAsRead(id);
        }
    },

    updateConversation: (conv: Conversation) => {
        set((state) => {
            const index = state.conversations.findIndex((c) => c.id === conv.id);
            if (index >= 0) {
                const newConversations = [...state.conversations];
                newConversations[index] = conv;
                // 按更新时间排序
                newConversations.sort((a, b) => b.updatedAt - a.updatedAt);
                return { conversations: newConversations };
            }
            // 新对话添加到顶部
            return { conversations: [conv, ...state.conversations] };
        });
    },

    // 从好友创建对话
    addConversationFromFriend: (friend: Friend) => {
        const existing = get().conversations.find((c) => c.id === friend.friendId);
        if (existing) {
            // 已存在，直接设为当前对话
            set({ activeConversationId: friend.friendId });
            return;
        }

        // 创建新对话
        const newConversation: Conversation = {
            id: friend.friendId,
            name: friend.remark || friend.nickname || friend.username,
            avatar: friend.avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${friend.friendId}`,
            lastMessage: '',
            unreadCount: 0,
            updatedAt: Date.now(),
        };

        set((state) => ({
            conversations: [newConversation, ...state.conversations],
            activeConversationId: friend.friendId,
        }));
    },

    removeConversation: (convId: string) => {
        set((state) => ({
            conversations: state.conversations.filter((c) => c.id !== convId),
            // 如果删除的是当前对话，清空选中
            activeConversationId: state.activeConversationId === convId ? null : state.activeConversationId,
        }));
    },

    markAsRead: (convId: string) => {
        set((state) => ({
            conversations: state.conversations.map((c) =>
                c.id === convId ? { ...c, unreadCount: 0 } : c
            ),
        }));
    },

    updateLastMessage: (convId: string, message: string) => {
        set((state) => ({
            conversations: state.conversations.map((c) =>
                c.id === convId
                    ? { ...c, lastMessage: message, updatedAt: Date.now() }
                    : c
            ).sort((a, b) => b.updatedAt - a.updatedAt),
        }));
    },
}));
