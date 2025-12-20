
import { Layout, Avatar, Input, Button } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useState, useMemo, useEffect } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useMessageStore } from '@/stores/messageStore';
import { useIMStore } from '@/stores/imStore';
import { messageDispatcher, ResponsePayload } from '@/services/messageDispatcher';
import styles from './Chat.module.css';

const { Sider, Content } = Layout;

// 稳定的空数组引用，避免每次渲染创建新引用导致无限循环
const EMPTY_MESSAGES: never[] = [];

function Chat() {
    const [inputValue, setInputValue] = useState('');
    const conversations = useChatStore((state) => state.conversations);
    const activeConversationId = useChatStore((state) => state.activeConversationId);
    const setActiveConversation = useChatStore((state) => state.setActiveConversation);

    // IM 连接状态
    const imStatus = useIMStore((state) => state.status);

    // 从 store 获取消息 Map
    const messagesMap = useMessageStore((state) => state.messages);
    const addMessage = useMessageStore((state) => state.addMessage);

    // 注册消息处理器
    useEffect(() => {
        const handleChatPush = (payload: Uint8Array | null, _reqId: string | null) => {
            if (payload) {
                // TODO: 解析 ChatPush payload 并添加消息
                console.log('[Chat] Received ChatPush');
            }
        };

        const handleChatSendAck = (payload: Uint8Array | null, reqId: string | null) => {
            console.log('[Chat] ChatSendAck for reqId:', reqId);
            // TODO: 更新消息状态
        };

        messageDispatcher.register(ResponsePayload.ChatPush, handleChatPush);
        messageDispatcher.register(ResponsePayload.ChatSendAck, handleChatSendAck);

        return () => {
            messageDispatcher.unregister(ResponsePayload.ChatPush, handleChatPush);
            messageDispatcher.unregister(ResponsePayload.ChatSendAck, handleChatSendAck);
        };
    }, [addMessage]);

    // 使用 useMemo 计算当前会话的消息，避免 selector 返回新引用
    const messages = useMemo(() => {
        if (!activeConversationId) return EMPTY_MESSAGES;
        return messagesMap.get(activeConversationId) ?? EMPTY_MESSAGES;
    }, [messagesMap, activeConversationId]);

    const sendMessage = useMessageStore((state) => state.sendMessage);

    const handleSend = () => {
        if (!inputValue.trim() || !activeConversationId) return;
        if (imStatus !== 'authenticated') {
            console.warn('[Chat] IM not authenticated, cannot send message');
            return;
        }
        sendMessage(activeConversationId, inputValue);
        setInputValue('');
    };

    return (
        <Layout className={styles.container}>
            <Sider width={300} className={styles.sider}>
                <div className={styles.siderHeader}>
                    <h3>会话</h3>
                    <span className={styles.status}>
                        {imStatus === 'authenticated' ? '🟢' : '🔴'} {'正常'}
                    </span>
                </div>
                <div className={styles.convList}>
                    {conversations.map((conv) => (
                        <div
                            key={conv.id}
                            className={`${styles.convItem} ${conv.id === activeConversationId ? styles.active : ''}`}
                            onClick={() => setActiveConversation(conv.id)}
                        >
                            <Avatar src={conv.avatar} className={styles.convAvatar} />
                            <div className={styles.convInfo}>
                                <div className={styles.convName}>{conv.name}</div>
                                <div className={styles.convLastMsg}>{conv.lastMessage}</div>
                            </div>
                        </div>
                    ))}
                </div>
            </Sider>
            <Content className={styles.content}>
                <div className={styles.messageList}>
                    {messages.map((msg) => (
                        <div key={msg.id} className={`${styles.message} ${msg.isSelf ? styles.self : ''}`}>
                            <div className={styles.bubble}>{msg.content}</div>
                        </div>
                    ))}
                </div>
                <div className={styles.inputArea}>
                    <Input
                        value={inputValue}
                        onChange={(e) => setInputValue(e.target.value)}
                        onPressEnter={handleSend}
                        placeholder="输入消息..."
                        size="large"
                        disabled={imStatus !== 'authenticated'}
                    />
                    <Button
                        type="primary"
                        icon={<SendOutlined />}
                        onClick={handleSend}
                        size="large"
                        disabled={imStatus !== 'authenticated'}
                    >
                        发送
                    </Button>
                </div>
            </Content>
        </Layout>
    );
}

export default Chat;
