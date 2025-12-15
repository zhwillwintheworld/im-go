import { Layout, List, Avatar, Input, Button } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useChatStore } from '@/stores/chatStore';
import { useMessageStore } from '@/stores/messageStore';
import styles from './Chat.module.css';

const { Sider, Content } = Layout;

function Chat() {
    const [inputValue, setInputValue] = useState('');
    const conversations = useChatStore((state) => state.conversations);
    const activeConversationId = useChatStore((state) => state.activeConversationId);
    const setActiveConversation = useChatStore((state) => state.setActiveConversation);
    const messages = useMessageStore((state) =>
        activeConversationId ? state.messages.get(activeConversationId) || [] : []
    );
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
                <List
                    dataSource={conversations}
                    renderItem={(conv) => (
                        <List.Item
                            className={`${styles.convItem} ${conv.id === activeConversationId ? styles.active : ''}`}
                            onClick={() => setActiveConversation(conv.id)}
                        >
                            <List.Item.Meta
                                avatar={<Avatar src={conv.avatar} />}
                                title={conv.name}
                                description={conv.lastMessage}
                            />
                        </List.Item>
                    )}
                />
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
