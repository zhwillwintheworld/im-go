import { create } from 'zustand';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol, FrameType } from '@/services/protocol/IMProtocol';
import { ChatType } from '@/protocol/im/protocol/chat-type';
import { MsgType } from '@/protocol/im/protocol/msg-type';
import { ResponsePayload } from '@/protocol/im/protocol/response-payload';

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
            // 使用新的 FlatBuffers 协议发送消息
            const { frame } = IMProtocol.createChatSendRequest(
                ChatType.PRIVATE,
                convId,  // targetId
                MsgType.TEXT,
                content
            );
            await transportManager.send(frame);
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
        transportManager.onMessage((frameType: FrameType, body: Uint8Array) => {
            console.log('Receive frame:', frameType);

            if (frameType === FrameType.Response) {
                const resp = IMProtocol.parseClientResponse(body);
                console.log('ClientResponse:', resp);

                switch (resp.payloadType) {
                    case ResponsePayload.ChatSendAck:
                        // 消息发送确认
                        if (resp.reqId) {
                            // TODO: 根据 reqId 更新消息状态
                            // 目前 reqId 与 msgId 不同，需要映射
                            console.log('ChatSendAck received for reqId:', resp.reqId);
                        }
                        break;
                    case ResponsePayload.ChatPush:
                        // 接收到新消息
                        if (resp.payload) {
                            // TODO: 解析 ChatPush payload
                            console.log('ChatPush received');
                        }
                        break;
                    default:
                        console.log('Unknown response payload type:', resp.payloadType);
                }
            }
        });
    }
}));
