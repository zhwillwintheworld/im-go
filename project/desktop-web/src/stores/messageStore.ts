import { create } from 'zustand';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol, MsgType } from '@/services/protocol/IMProtocol';

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
    initListener: () => void;
}

export const useMessageStore = create<MessageState>((set, get) => ({
    messages: new Map(),

    addMessage: (convId: string, msg: Message) => {
        set((state) => {
            const newMessages = new Map(state.messages);
            const convMessages = newMessages.get(convId) || [];
            // 避免重复添加
            if (!convMessages.find(m => m.id === msg.id)) {
                newMessages.set(convId, [...convMessages, msg]);
            }
            return { messages: newMessages };
        });
    },

    sendMessage: async (convId: string, content: string) => {
        const msgId = crypto.randomUUID();
        // 假设 convId 就是对方 userId (单聊)
        // 实际项目应区分单聊/群聊
        const toUserId = parseInt(convId);

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

        try {
            // 构建客户端消息
            const clientMsg = {
                clientMsgId: msgId,
                toUserId: toUserId,
                toGroupId: 0,
                content: content // 直接发送字符串内容
            };

            const bytes = IMProtocol.encode(MsgType.Message, clientMsg);
            await transportManager.send(bytes);
        } catch (e) {
            console.error('Failed to send message:', e);
            get().updateMessageStatus(msgId, 'failed');
        }
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

    initListener: () => {
        transportManager.onMessage((buffer) => {
            const packet = IMProtocol.decode(buffer);
            if (!packet) return;

            console.log('Receive packet:', packet);

            switch (packet.msgType) {
                case MsgType.MessageAck:
                    const ack = packet.body;
                    if (ack.ClientMsgId) {
                        get().updateMessageStatus(ack.ClientMsgId, 'sent');
                    }
                    break;
                case MsgType.Message:
                    const pushMsg = packet.body;
                    // 处理接收到的消息
                    // 假设 FromUserId 是发送者 ID，也是 conversationId
                    const convId = pushMsg.FromUserId.toString();

                    // Content 是 Base64 编码的字符串，需要解码
                    let content = '';
                    try {
                        const binaryStr = atob(pushMsg.Content);
                        const bytes = new Uint8Array(binaryStr.length);
                        for (let i = 0; i < binaryStr.length; i++) {
                            bytes[i] = binaryStr.charCodeAt(i);
                        }
                        content = new TextDecoder().decode(bytes);
                    } catch (e) {
                        console.error('Failed to decode message content:', e);
                        content = '[Decoding Error]';
                    }

                    const newMsg: Message = {
                        id: pushMsg.ServerMsgId.toString(),
                        conversationId: convId,
                        content: content,
                        senderId: pushMsg.FromUserId.toString(),
                        isSelf: false,
                        timestamp: pushMsg.Timestamp,
                        status: 'sent'
                    };
                    get().addMessage(convId, newMsg);
                    break;
            }
        });
    }
}));

