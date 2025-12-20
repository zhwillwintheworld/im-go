-- ============================================
-- IM 系统 PostgreSQL 测试数据
-- ============================================
-- 使用方式: 在 start-postgres.sh 中自动执行
-- 注意: id 使用雪花ID格式，由应用层生成
-- ============================================

-- 插入测试用户
-- 密码: 123456 (bcrypt hash)
INSERT INTO users (id, username, password_hash, nickname, avatar, status) VALUES
(1000000000000001, 'alice', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Alice', 'https://api.dicebear.com/7.x/avataaars/svg?seed=alice', 0),
(1000000000000002, 'bob', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Bob', 'https://api.dicebear.com/7.x/avataaars/svg?seed=bob', 0),
(1000000000000003, 'charlie', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Charlie', 'https://api.dicebear.com/7.x/avataaars/svg?seed=charlie', 0),
(1000000000000004, 'david', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'David', 'https://api.dicebear.com/7.x/avataaars/svg?seed=david', 0),
(1000000000000005, 'eve', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Eve', 'https://api.dicebear.com/7.x/avataaars/svg?seed=eve', 0);

-- 插入好友关系 (双向)
-- Alice(1000000000000001) <-> Bob(1000000000000002), Charlie(1000000000000003), Eve(1000000000000005)
-- Bob(1000000000000002) <-> Charlie(1000000000000003), David(1000000000000004)
-- Charlie(1000000000000003) <-> David(1000000000000004)
INSERT INTO friends (id, user_id, friend_id, remark) VALUES
-- Alice 的好友
(2000000000000001, 1000000000000001, 1000000000000002, 'Bob'),
(2000000000000002, 1000000000000002, 1000000000000001, 'Alice'),
(2000000000000003, 1000000000000001, 1000000000000003, 'Charlie'),
(2000000000000004, 1000000000000003, 1000000000000001, 'Alice'),
(2000000000000005, 1000000000000001, 1000000000000005, 'Eve'),
(2000000000000006, 1000000000000005, 1000000000000001, 'Alice'),
-- Bob 的好友
(2000000000000007, 1000000000000002, 1000000000000003, ''),
(2000000000000008, 1000000000000003, 1000000000000002, ''),
(2000000000000009, 1000000000000002, 1000000000000004, 'Dave'),
(2000000000000010, 1000000000000004, 1000000000000002, 'Bob'),
-- Charlie 和 David
(2000000000000011, 1000000000000003, 1000000000000004, ''),
(2000000000000012, 1000000000000004, 1000000000000003, '');
