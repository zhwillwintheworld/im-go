import { Layout, Avatar, Input, Button, Tabs, Empty } from 'antd';
import { SendOutlined, MessageOutlined, TeamOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { useState, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useChatStore } from '@/stores/chatStore';
import { useMessageStore } from '@/stores/messageStore';
import { useIMStore } from '@/stores/imStore';
import FriendList from '@/components/FriendList';
import GroupList from '@/components/GroupList';
import styles from './Chat.module.css';

const { Sider, Content } = Layout;

// 稳定的空数组引用，避免每次渲染创建新引用导致无限循环
const EMPTY_MESSAGES: never[] = [];

function Chat() {
    const navigate = useNavigate();
    const [inputValue, setInputValue] = useState('');
    const [activeTab, setActiveTab] = useState<string>('chats');
    const conversations = useChatStore((state) => state.conversations);
    const activeConversationId = useChatStore((state) => state.activeConversationId);
    const setActiveConversation = useChatStore((state) => state.setActiveConversation);

    // IM 连接状态
    const imStatus = useIMStore((state) => state.status);

    // 从 store 获取消息 Map
    const messagesMap = useMessageStore((state) => state.messages);
    const initListener = useMessageStore((state) => state.initListener);

    // 初始化消息监听器
    useEffect(() => {
        initListener();
    }, [initListener]);

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

    // 选择好友开始聊天后切换到会话 tab
    const handleStartChat = () => {
        setActiveTab('chats');
    };

    // 渲染会话列表
    const renderConversationList = () => {
        if (conversations.length === 0) {
            return (
                <div className={styles.emptyList}>
                    <Empty
                        description="暂无会话"
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                    >
                        <Button type="link" onClick={() => setActiveTab('friends')}>
                            去添加好友开始聊天
                        </Button>
                    </Empty>
                </div>
            );
        }

        return (
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
                            <div className={styles.convLastMsg}>{conv.lastMessage || '暂无消息'}</div>
                        </div>
                        {conv.unreadCount > 0 && (
                            <span className={styles.unreadBadge}>{conv.unreadCount}</span>
                        )}
                    </div>
                ))}
            </div>
        );
    };

    const tabItems = [
        {
            key: 'chats',
            label: (
                <span>
                    <MessageOutlined />
                    会话
                </span>
            ),
            children: renderConversationList(),
        },
        {
            key: 'friends',
            label: (
                <span>
                    <TeamOutlined />
                    好友
                </span>
            ),
            children: <FriendList onStartChat={handleStartChat} />,
        },
        {
            key: 'groups',
            label: (
                <span>
                    <TeamOutlined />
                    群组
                </span>
            ),
            children: <GroupList onStartChat={handleStartChat} />,
        },
    ];

    return (
        <Layout className={styles.container}>
            <Sider width={300} className={styles.sider}>
                <div className={styles.siderHeader}>
                    <div className={styles.backBtn} onClick={() => navigate('/home')}>
                        <ArrowLeftOutlined /> 主页
                    </div>
                    <span className={styles.status}>
                        {imStatus === 'authenticated' ? '🟢' : '🔴'} {imStatus === 'authenticated' ? '在线' : '离线'}
                    </span>
                </div>
                <Tabs
                    activeKey={activeTab}
                    onChange={setActiveTab}
                    items={tabItems}
                    centered
                    destroyOnHidden
                    className={styles.tabs}
                />
            </Sider>
            <Content className={styles.content}>
                {activeConversationId ? (
                    <>
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
                    </>
                ) : (
                    <div className={styles.noConversation}>
                        <Empty description="选择一个会话开始聊天" />
                    </div>
                )}
            </Content>
        </Layout>
    );
}

export default Chat;
