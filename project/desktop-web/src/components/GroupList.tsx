import React from 'react';
import { Avatar, Empty, Button } from 'antd';
import { TeamOutlined, MessageOutlined } from '@ant-design/icons';
import { useChatStore } from '../stores/chatStore';
import './GroupList.css';

// 群组类型定义
export interface Group {
    id: string;
    name: string;
    avatar?: string;
    memberCount: number;
}

interface GroupListProps {
    onStartChat?: (group: Group) => void;
}

const GroupList: React.FC<GroupListProps> = ({ onStartChat }) => {
    const { updateConversation, setActiveConversation } = useChatStore();

    // TODO: 后续从 API 获取群组列表，目前使用空数组占位
    const groups: Group[] = [];

    const handleStartGroupChat = (group: Group) => {
        const convId = `group_${group.id}`;
        // 添加群组会话
        updateConversation({
            id: convId,
            name: group.name,
            avatar: group.avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${group.id}`,
            lastMessage: '',
            unreadCount: 0,
            updatedAt: Date.now(),
        });
        setActiveConversation(convId);
        onStartChat?.(group);
    };

    if (groups.length === 0) {
        return (
            <div className="group-list-empty">
                <Empty description="暂无群组" />
            </div>
        );
    }

    return (
        <div className="group-list">
            {groups.map((group) => (
                <div key={group.id} className="group-item">
                    <Avatar
                        src={group.avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${group.id}`}
                        icon={<TeamOutlined />}
                        className="group-avatar"
                    />
                    <div className="group-info">
                        <div className="group-name">{group.name}</div>
                        <div className="group-members">{group.memberCount} 成员</div>
                    </div>
                    <div className="group-actions">
                        <Button
                            type="text"
                            icon={<MessageOutlined />}
                            onClick={() => handleStartGroupChat(group)}
                            title="发消息"
                        />
                    </div>
                </div>
            ))}
        </div>
    );
};

export default GroupList;
