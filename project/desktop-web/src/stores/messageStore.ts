import { create } from 'zustand';

interface Message {
    id: string;
    conversationId: string;
    content: string;
    senderId: string;
    isSelf: boolean;
    timestamp: number;
    status: 'pending' | 'sent' | 'failed';
}

interface MessageState {
    messages: Map<string, Message[]>;
    addMessage: (convId: string, msg: Message) => void;
    sendMessage: (convId: string, content: string) => Promise<void>;
    updateMessageStatus: (msgId: string, status: Message['status']) => void;
}

export const useMessageStore = create<MessageState>((set, get) => ({
    messages: new Map(),

    addMessage: (convId: string, msg: Message) => {
        set((state) => {
            const newMessages = new Map(state.messages);
            const convMessages = newMessages.get(convId) || [];
            newMessages.set(convId, [...convMessages, msg]);
            return { messages: newMessages };
        });
    },

    sendMessage: async (convId: string, content: string) => {
        const msgId = crypto.randomUUID();
        const msg: Message = {
            id: msgId,
            conversationId: convId,
            content,
            senderId: 'self',
            isSelf: true,
            timestamp: Date.now(),
            status: 'pending',
        };

        get().addMessage(convId, msg);

        // TODO: 通过 WebSocket 发送消息
        // 模拟发送成功
        setTimeout(() => {
            get().updateMessageStatus(msgId, 'sent');
        }, 500);
    },

    updateMessageStatus: (msgId: string, status: Message['status']) => {
        set((state) => {
            const newMessages = new Map(state.messages);
            for (const [convId, msgs] of newMessages) {
                const index = msgs.findIndex((m) => m.id === msgId);
                if (index >= 0) {
                    const newMsgs = [...msgs];
                    newMsgs[index] = { ...newMsgs[index], status };
                    newMessages.set(convId, newMsgs);
                    break;
                }
            }
            return { messages: newMessages };
        });
    },
}));
