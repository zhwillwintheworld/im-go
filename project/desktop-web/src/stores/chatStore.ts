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
    conversations: [],
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
