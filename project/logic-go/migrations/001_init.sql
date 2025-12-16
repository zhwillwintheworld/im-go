-- Logic Layer 数据库迁移脚本
-- 创建 IM 系统所需的表

-- 1. 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    nickname VARCHAR(128),
    avatar VARCHAR(512),
    status INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);

-- 2. 消息表
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    client_msg_id VARCHAR(64) NOT NULL,
    from_user_id BIGINT NOT NULL,
    to_user_id BIGINT,
    to_group_id BIGINT,
    msg_type INT NOT NULL DEFAULT 1,
    content BYTEA,
    status INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_from_user ON messages(from_user_id, created_at DESC);
CREATE INDEX idx_messages_to_user ON messages(to_user_id, created_at DESC) WHERE to_user_id IS NOT NULL;
CREATE INDEX idx_messages_to_group ON messages(to_group_id, created_at DESC) WHERE to_group_id IS NOT NULL;
CREATE INDEX idx_messages_client_msg_id ON messages(client_msg_id);

-- 3. 群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    owner_id BIGINT NOT NULL REFERENCES users(id),
    avatar VARCHAR(512),
    description TEXT,
    member_count INT NOT NULL DEFAULT 0,
    status INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_groups_owner ON groups(owner_id);

-- 4. 群成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role INT NOT NULL DEFAULT 0,  -- 0: 成员, 1: 管理员, 2: 群主
    nickname VARCHAR(128),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_group_members_user ON group_members(user_id);
CREATE INDEX idx_group_members_group ON group_members(group_id);

-- 5. 离线消息表
CREATE TABLE IF NOT EXISTS offline_messages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, message_id)
);

CREATE INDEX idx_offline_messages_user ON offline_messages(user_id, created_at ASC);

-- 6. 会话表 (可选，用于会话列表)
CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    peer_user_id BIGINT,
    group_id BIGINT,
    last_message_id BIGINT REFERENCES messages(id),
    last_read_message_id BIGINT,
    unread_count INT NOT NULL DEFAULT 0,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, peer_user_id) WHERE peer_user_id IS NOT NULL,
    UNIQUE(user_id, group_id) WHERE group_id IS NOT NULL
);

CREATE INDEX idx_conversations_user ON conversations(user_id, updated_at DESC);

-- 触发器：自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_group_members_updated_at
    BEFORE UPDATE ON group_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 触发器：自动更新群成员数量
CREATE OR REPLACE FUNCTION update_group_member_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE groups SET member_count = member_count + 1 WHERE id = NEW.group_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE groups SET member_count = member_count - 1 WHERE id = OLD.group_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_member_count
    AFTER INSERT OR DELETE ON group_members
    FOR EACH ROW EXECUTE FUNCTION update_group_member_count();
