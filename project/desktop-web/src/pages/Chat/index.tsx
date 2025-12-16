import { Layout, Avatar, Input, Button } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useState, useMemo, useEffect } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useMessageStore } from '@/stores/messageStore';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol, MsgType } from '@/services/protocol/IMProtocol';
import styles from './Chat.module.css';

const { Sider, Content } = Layout;

// 稳定的空数组引用，避免每次渲染创建新引用导致无限循环
const EMPTY_MESSAGES: never[] = [];

function Chat() {
    const [inputValue, setInputValue] = useState('');
    const conversations = useChatStore((state) => state.conversations);
    const activeConversationId = useChatStore((state) => state.activeConversationId);
    const setActiveConversation = useChatStore((state) => state.setActiveConversation);

    // 从 store 获取消息 Map
    const messagesMap = useMessageStore((state) => state.messages);
    const initListener = useMessageStore((state) => state.initListener);

    // 初始化连接
    useEffect(() => {
        const connect = async () => {
            try {
                // 开发环境自签名证书需要 Chrome 启动参数 --origin-to-force-quic-on=localhost:8443
                await transportManager.connect('https://localhost:8443/webtransport');

                // 发送认证请求 (内容暂时随意，服务端目前 hardcode 用户 ID 为 1)
                const authBytes = IMProtocol.encode(MsgType.Auth, { token: "mock-token" });
                await transportManager.send(authBytes);

                console.log('Connected and Authenticated');
                initListener();
            } catch (err) {
                console.error('Failed to connect:', err);
            }
        };

        connect();

        return () => {
            transportManager.disconnect();
        };
    }, [initListener]);

    // 使用 useMemo 计算当前会话的消息，避免 selector 返回新引用
    const messages = useMemo(() => {
        if (!activeConversationId) return EMPTY_MESSAGES;
        return messagesMap.get(activeConversationId) ?? EMPTY_MESSAGES;
    }, [messagesMap, activeConversationId]);

    const sendMessage = useMessageStore((state) => state.sendMessage);

    const handleSend = () => {
        if (!inputValue.trim() || !activeConversationId) return;
        sendMessage(activeConversationId, inputValue);
        setInputValue('');
    };

    return (
        <Layout className={styles.container}>
            <Sider width={300} className={styles.sider}>
                <div className={styles.siderHeader}>
                    <h3>会话</h3>
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
                    />
                    <Button
                        type="primary"
                        icon={<SendOutlined />}
                        onClick={handleSend}
                        size="large"
                    >
                        发送
                    </Button>
                </div>
            </Content>
        </Layout>
    );
}

export default Chat;
