-- ============================================
-- IM 系统 PostgreSQL Schema
-- ============================================
-- 使用方式: 在 start-postgres.sh 中自动执行
-- ============================================

-- ============================================
-- 数据库级别字符集设置 (支持表情符号)
-- ============================================
-- PostgreSQL 使用 UTF8 编码完整支持 emoji 表情
-- 如需手动创建数据库，请使用以下命令:
--
-- CREATE DATABASE im_db
--     WITH ENCODING = 'UTF8'
--     LC_COLLATE = 'en_US.UTF-8'
--     LC_CTYPE = 'en_US.UTF-8'
--     TEMPLATE = template0;
--
-- 或使用 ICU collation (PostgreSQL 15+):
-- CREATE DATABASE im_db
--     WITH ENCODING = 'UTF8'
--     LOCALE_PROVIDER = icu
--     ICU_LOCALE = 'und-u-ks-level2'
--     TEMPLATE = template0;
-- ============================================

-- 设置客户端编码为 UTF8
SET client_encoding = 'UTF8';

-- ============================================
-- 建表规范:
-- 1. 每个表必须包含以下标准字段:
--    - id: BIGINT PRIMARY KEY (雪花ID，对外唯一标识)
--    - create_at: TIMESTAMP WITH TIME ZONE (创建时间)
--    - update_at: TIMESTAMP WITH TIME ZONE (更新时间)
--    - deleted: INT (逻辑删除: 0=正常, 1=已删除)
-- 2. id 使用雪花算法生成，由应用层负责生成，不使用自增
-- 3. 不使用外键约束，数据完整性在应用层保证
-- 4. 每个字段必须有字段描述
-- 5. 字符串字段设置 NOT NULL 且默认值为空字符串
-- 6. update_at 和 member_count 等字段由应用层维护，不使用触发器
-- ============================================

-- 删除已存在的表
DROP TABLE IF EXISTS group_members CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS friends CASCADE;
DROP TABLE IF EXISTS friend_requests CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- 1. 用户表
CREATE TABLE users (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    username VARCHAR(64) NOT NULL UNIQUE,                               -- 用户名，唯一
    password_hash VARCHAR(256) NOT NULL DEFAULT '',                     -- 密码哈希值
    nickname VARCHAR(128) NOT NULL DEFAULT '',                          -- 用户昵称
    avatar VARCHAR(512) NOT NULL DEFAULT '',                            -- 头像URL
    status INT NOT NULL DEFAULT 0,                                      -- 状态: 0=正常, 1=禁用
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0                                      -- 逻辑删除: 0=正常, 1=已删除
);

CREATE INDEX idx_users_username ON users(username);

COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users.id IS '雪花ID，主键';
COMMENT ON COLUMN users.username IS '用户名，唯一';
COMMENT ON COLUMN users.password_hash IS '密码哈希值';
COMMENT ON COLUMN users.nickname IS '用户昵称';
COMMENT ON COLUMN users.avatar IS '头像URL';
COMMENT ON COLUMN users.status IS '状态: 0=正常, 1=禁用';
COMMENT ON COLUMN users.create_at IS '创建时间';
COMMENT ON COLUMN users.update_at IS '更新时间';
COMMENT ON COLUMN users.deleted IS '逻辑删除: 0=正常, 1=已删除';

-- 2. 好友邀请表
CREATE TABLE friend_requests (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    from_user_id BIGINT NOT NULL,                                       -- 发起者用户ID，关联users.id
    to_user_id BIGINT NOT NULL,                                         -- 接收者用户ID，关联users.id
    message VARCHAR(256) NOT NULL DEFAULT '',                           -- 邀请附言
    status INT NOT NULL DEFAULT 0,                                      -- 状态: 0=待处理, 1=已同意, 2=已拒绝
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0                                      -- 逻辑删除: 0=正常, 1=已删除
);

CREATE INDEX idx_friend_requests_from_user ON friend_requests(from_user_id);
CREATE INDEX idx_friend_requests_to_user ON friend_requests(to_user_id);

COMMENT ON TABLE friend_requests IS '好友邀请表';
COMMENT ON COLUMN friend_requests.id IS '雪花ID，主键';
COMMENT ON COLUMN friend_requests.from_user_id IS '发起者用户ID，关联users.id';
COMMENT ON COLUMN friend_requests.to_user_id IS '接收者用户ID，关联users.id';
COMMENT ON COLUMN friend_requests.message IS '邀请附言';
COMMENT ON COLUMN friend_requests.status IS '状态: 0=待处理, 1=已同意, 2=已拒绝';
COMMENT ON COLUMN friend_requests.create_at IS '创建时间';
COMMENT ON COLUMN friend_requests.update_at IS '更新时间';
COMMENT ON COLUMN friend_requests.deleted IS '逻辑删除: 0=正常, 1=已删除';

-- 3. 好友关系表（只存储已确认的好友关系）
CREATE TABLE friends (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    user_id BIGINT NOT NULL,                                            -- 用户ID，关联users.id
    friend_id BIGINT NOT NULL,                                          -- 好友用户ID，关联users.id
    remark VARCHAR(128) NOT NULL DEFAULT '',                            -- 好友备注
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0,                                     -- 逻辑删除: 0=正常, 1=已删除
    UNIQUE(user_id, friend_id)
);

CREATE INDEX idx_friends_user ON friends(user_id);
CREATE INDEX idx_friends_friend ON friends(friend_id);

COMMENT ON TABLE friends IS '好友关系表（只存储已确认的好友关系）';
COMMENT ON COLUMN friends.id IS '雪花ID，主键';
COMMENT ON COLUMN friends.user_id IS '用户ID，关联users.id';
COMMENT ON COLUMN friends.friend_id IS '好友用户ID，关联users.id';
COMMENT ON COLUMN friends.remark IS '好友备注';
COMMENT ON COLUMN friends.create_at IS '创建时间';
COMMENT ON COLUMN friends.update_at IS '更建时间';
COMMENT ON COLUMN friends.deleted IS '逻辑删除: 0=正常, 1=已删除';

-- 4. 消息表
CREATE TABLE messages (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    client_msg_id VARCHAR(64) NOT NULL DEFAULT '',                      -- 客户端消息ID，用于去重
    from_user_id BIGINT NOT NULL,                                       -- 发送者用户ID，关联users.id
    to_user_id BIGINT,                                                  -- 接收者用户ID，私聊时使用，关联users.id
    to_group_id BIGINT,                                                 -- 接收群组ID，群聊时使用，关联groups.id
    msg_type INT NOT NULL DEFAULT 1,                                    -- 消息类型: 1=文本, 2=图片, 3=语音, 4=视频, 5=文件
    content BYTEA,                                                      -- 消息内容，二进制存储
    status INT NOT NULL DEFAULT 0,                                      -- 状态: 0=正常, 1=已撤回, 2=已删除
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0                                      -- 逻辑删除: 0=正常, 1=已删除
);

CREATE INDEX idx_messages_from_user ON messages(from_user_id, create_at DESC);
CREATE INDEX idx_messages_to_user ON messages(to_user_id, create_at DESC) WHERE to_user_id IS NOT NULL;
CREATE INDEX idx_messages_to_group ON messages(to_group_id, create_at DESC) WHERE to_group_id IS NOT NULL;
CREATE INDEX idx_messages_client_msg_id ON messages(client_msg_id);

COMMENT ON TABLE messages IS '消息表';
COMMENT ON COLUMN messages.id IS '雪花ID，主键';
COMMENT ON COLUMN messages.client_msg_id IS '客户端消息ID，用于去重';
COMMENT ON COLUMN messages.from_user_id IS '发送者用户ID，关联users.id';
COMMENT ON COLUMN messages.to_user_id IS '接收者用户ID，私聊时使用，关联users.id';
COMMENT ON COLUMN messages.to_group_id IS '接收群组ID，群聊时使用，关联groups.id';
COMMENT ON COLUMN messages.msg_type IS '消息类型: 1=文本, 2=图片, 3=语音, 4=视频, 5=文件';
COMMENT ON COLUMN messages.content IS '消息内容，二进制存储';
COMMENT ON COLUMN messages.status IS '状态: 0=正常, 1=已撤回, 2=已删除';
COMMENT ON COLUMN messages.create_at IS '创建时间';
COMMENT ON COLUMN messages.update_at IS '更新时间';
COMMENT ON COLUMN messages.deleted IS '逻辑删除: 0=正常, 1=已删除';

-- 5. 群组表
CREATE TABLE groups (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    name VARCHAR(128) NOT NULL DEFAULT '',                              -- 群组名称
    owner_id BIGINT NOT NULL,                                           -- 群主用户ID，关联users.id
    avatar VARCHAR(512) NOT NULL DEFAULT '',                            -- 群头像URL
    description VARCHAR(1024) NOT NULL DEFAULT '',                      -- 群描述
    max_members INT NOT NULL DEFAULT 200,                               -- 最大成员数量
    status INT NOT NULL DEFAULT 0,                                      -- 状态: 0=正常, 1=解散
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0                                      -- 逻辑删除: 0=正常, 1=已删除
);

CREATE INDEX idx_groups_owner ON groups(owner_id);

COMMENT ON TABLE groups IS '群组表';
COMMENT ON COLUMN groups.id IS '雪花ID，主键';
COMMENT ON COLUMN groups.name IS '群组名称';
COMMENT ON COLUMN groups.owner_id IS '群主用户ID，关联users.id';
COMMENT ON COLUMN groups.avatar IS '群头像URL';
COMMENT ON COLUMN groups.description IS '群描述';
COMMENT ON COLUMN groups.max_members IS '最大成员数量';
COMMENT ON COLUMN groups.status IS '状态: 0=正常, 1=解散';
COMMENT ON COLUMN groups.create_at IS '创建时间';
COMMENT ON COLUMN groups.update_at IS '更新时间';
COMMENT ON COLUMN groups.deleted IS '逻辑删除: 0=正常, 1=已删除';

-- 6. 群成员表
CREATE TABLE group_members (
    id BIGINT PRIMARY KEY,                                              -- 雪花ID，主键
    group_id BIGINT NOT NULL,                                           -- 群组ID，关联groups.id
    user_id BIGINT NOT NULL,                                            -- 用户ID，关联users.id
    role INT NOT NULL DEFAULT 0,                                        -- 角色: 0=成员, 1=管理员, 2=群主
    nickname VARCHAR(128) NOT NULL DEFAULT '',                          -- 群内昵称
    create_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 创建时间
    update_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),          -- 更新时间
    deleted INT NOT NULL DEFAULT 0,                                     -- 逻辑删除: 0=正常, 1=已删除
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_group_members_user ON group_members(user_id);
CREATE INDEX idx_group_members_group ON group_members(group_id);

COMMENT ON TABLE group_members IS '群成员表';
COMMENT ON COLUMN group_members.id IS '雪花ID，主键';
COMMENT ON COLUMN group_members.group_id IS '群组ID，关联groups.id';
COMMENT ON COLUMN group_members.user_id IS '用户ID，关联users.id';
COMMENT ON COLUMN group_members.role IS '角色: 0=成员, 1=管理员, 2=群主';
COMMENT ON COLUMN group_members.nickname IS '群内昵称';
COMMENT ON COLUMN group_members.create_at IS '创建时间';
COMMENT ON COLUMN group_members.update_at IS '更新时间';
COMMENT ON COLUMN group_members.deleted IS '逻辑删除: 0=正常, 1=已删除';

