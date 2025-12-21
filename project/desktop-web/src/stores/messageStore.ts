import { create } from 'zustand';
import * as flatbuffers from 'flatbuffers';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol, FrameType } from '@/services/protocol/IMProtocol';
import { ChatType } from '@/protocol/im/protocol/chat-type';
import { MsgType } from '@/protocol/im/protocol/msg-type';
import { ResponsePayload } from '@/protocol/im/protocol/response-payload';
import { ChatPush } from '@/protocol/im/protocol/chat-push';
import { useChatStore } from './chatStore';

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
    handleChatPush: (payload: Uint8Array) => void;
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

            // 更新会话最后消息
            useChatStore.getState().updateLastMessage(convId, content);
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

    // 处理收到的 ChatPush 消息
    handleChatPush: (payload: Uint8Array) => {
        try {
            const bb = new flatbuffers.ByteBuffer(payload);
            const chatPush = ChatPush.getRootAsChatPush(bb);

            const msgId = chatPush.msgId() || '';
            const senderId = chatPush.senderId() || '';
            const content = chatPush.content() || '';
            const sendTime = chatPush.sendTime();

            console.log('[MessageStore] ChatPush parsed:', {
                msgId,
                senderId,
                content: content.length > 50 ? content.substring(0, 50) + '...' : content,
                sendTime: sendTime.toString()
            });

            // 会话 ID 使用发送者 ID（私聊场景）
            const conversationId = senderId;

            // 创建消息对象
            const msg: Message = {
                id: msgId,
                conversationId,
                content,
                senderId,
                isSelf: false,
                timestamp: Number(sendTime),
                status: 'sent',
            };

            // 添加到消息历史
            get().addMessage(conversationId, msg);

            // 更新会话列表
            const chatStore = useChatStore.getState();
            const existingConv = chatStore.conversations.find(c => c.id === conversationId);

            if (existingConv) {
                // 已有会话，更新最后消息和未读数
                chatStore.updateConversation({
                    ...existingConv,
                    lastMessage: content,
                    unreadCount: chatStore.activeConversationId === conversationId
                        ? 0  // 当前正在查看的会话不增加未读
                        : existingConv.unreadCount + 1,
                    updatedAt: Number(sendTime),
                });
            } else {
                // 新会话，创建并添加
                chatStore.updateConversation({
                    id: conversationId,
                    name: senderId,  // 暂时使用 senderId 作为名称，后续可从用户信息获取
                    avatar: `https://api.dicebear.com/7.x/avataaars/svg?seed=${senderId}`,
                    lastMessage: content,
                    unreadCount: 1,
                    updatedAt: Number(sendTime),
                });
            }
        } catch (e) {
            console.error('[MessageStore] Failed to parse ChatPush:', e);
        }
    },

    initListener: () => {
        console.log('[MessageStore] initListener called, registering message handler');
        transportManager.onMessage((frameType: FrameType, body: Uint8Array) => {
            if (frameType === FrameType.Response) {
                const resp = IMProtocol.parseClientResponse(body);
                console.log('[MessageStore] ClientResponse:', {
                    reqId: resp.reqId,
                    code: resp.code,
                    msg: resp.msg,
                    payloadType: resp.payloadType,
                    payloadLength: resp.payload?.length || 0
                });

                switch (resp.payloadType) {
                    case ResponsePayload.ChatSendAck:
                        // 消息发送确认
                        if (resp.reqId) {
                            // TODO: 根据 reqId 更新消息状态
                            // 目前 reqId 与 msgId 不同，需要映射
                            console.log('[MessageStore] ChatSendAck received for reqId:', resp.reqId);
                        }
                        break;
                    case ResponsePayload.ChatPush:
                        if (resp.payload) {
                            get().handleChatPush(resp.payload);
                        } else {
                            console.warn('[MessageStore] ChatPush has no payload!');
                        }
                        break;
                    default:
                        console.log('[MessageStore] Unknown response payload type:', resp.payloadType);
                }
            } else {
                console.log('[MessageStore] Non-Response frame type:', frameType);
            }
        });
    }
}));
