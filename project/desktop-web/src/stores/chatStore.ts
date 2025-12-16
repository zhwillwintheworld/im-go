import { create } from 'zustand';

interface Conversation {
    id: string;
    name: string;
    avatar: string;
    lastMessage: string;
    unreadCount: number;
    updatedAt: number;
}

interface ChatState {
    conversations: Conversation[];
    activeConversationId: string | null;
    setActiveConversation: (id: string) => void;
    updateConversation: (conv: Conversation) => void;
    markAsRead: (convId: string) => void;
}

export const useChatStore = create<ChatState>((set) => ({
    conversations: [
        {
            id: '2',
            name: '用户二',
            avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=2',
            lastMessage: '你好',
            unreadCount: 0,
            updatedAt: Date.now(),
        },
        {
            id: '3',
            name: '用户三',
            avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=3',
            lastMessage: '在吗？',
            unreadCount: 1,
            updatedAt: Date.now(),
        }
    ],
    activeConversationId: null,

    setActiveConversation: (id: string) => {
        set({ activeConversationId: id });
    },

    updateConversation: (conv: Conversation) => {
        set((state) => {
            const index = state.conversations.findIndex((c) => c.id === conv.id);
            if (index >= 0) {
                const newConversations = [...state.conversations];
                newConversations[index] = conv;
                return { conversations: newConversations };
            }
            return { conversations: [conv, ...state.conversations] };
        });
    },

    markAsRead: (convId: string) => {
        set((state) => ({
            conversations: state.conversations.map((c) =>
                c.id === convId ? { ...c, unreadCount: 0 } : c
            ),
        }));
    },
}));
