import React, { useEffect } from 'react';
import { Avatar, Empty, Spin, Button, Popconfirm, message } from 'antd';
import { UserOutlined, MessageOutlined, DeleteOutlined } from '@ant-design/icons';
import { useFriendStore, Friend } from '../stores/friendStore';
import { useChatStore } from '../stores/chatStore';
import './FriendList.css';

interface FriendListProps {
    onStartChat?: (friend: Friend) => void;
}

const FriendList: React.FC<FriendListProps> = ({ onStartChat }) => {
    const { friends, loading, fetchFriends, deleteFriend } = useFriendStore();
    const { addConversationFromFriend } = useChatStore();

    useEffect(() => {
        fetchFriends();
    }, [fetchFriends]);

    const handleStartChat = (friend: Friend) => {
        addConversationFromFriend(friend);
        onStartChat?.(friend);
    };

    const handleDeleteFriend = async (friendId: string) => {
        const success = await deleteFriend(friendId);
        if (success) {
            message.success('已删除好友');
        }
    };

    if (loading) {
        return (
            <div className="friend-list-loading">
                <Spin />
            </div>
        );
    }

    // 确保 friends 是数组
    const friendList = Array.isArray(friends) ? friends : [];

    if (friendList.length === 0) {
        return (
            <div className="friend-list-empty">
                <Empty description="暂无好友" />
            </div>
        );
    }

    return (
        <div className="friend-list">
            {friendList.map((friend) => (
                <div key={friend.id} className="friend-item">
                    <Avatar
                        src={friend.avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${friend.friend_id}`}
                        icon={<UserOutlined />}
                        className="friend-avatar"
                    />
                    <div className="friend-info">
                        <div className="friend-name">
                            {friend.remark || friend.nickname || friend.username}
                        </div>
                        <div className="friend-username">@{friend.username}</div>
                    </div>
                    <div className="friend-actions">
                        <Button
                            type="text"
                            icon={<MessageOutlined />}
                            onClick={() => handleStartChat(friend)}
                            title="发消息"
                        />
                        <Popconfirm
                            title="确定删除该好友？"
                            onConfirm={() => handleDeleteFriend(friend.friend_id)}
                            okText="确定"
                            cancelText="取消"
                        >
                            <Button
                                type="text"
                                danger
                                icon={<DeleteOutlined />}
                                title="删除好友"
                            />
                        </Popconfirm>
                    </div>
                </div>
            ))}
        </div>
    );
};

export default FriendList;
