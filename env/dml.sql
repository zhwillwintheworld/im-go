-- ============================================
-- IM 系统 PostgreSQL 测试数据
-- ============================================
-- 使用方式: 在 start-postgres.sh 中自动执行
-- ============================================

-- 插入测试用户
-- 密码: 123456 (bcrypt hash)
INSERT INTO users (id, object_code, username, password_hash, nickname, avatar, status) VALUES
(1, '1000000000000001', 'alice', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Alice', 'https://api.dicebear.com/7.x/avataaars/svg?seed=alice', 0),
(2, '1000000000000002', 'bob', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Bob', 'https://api.dicebear.com/7.x/avataaars/svg?seed=bob', 0),
(3, '1000000000000003', 'charlie', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Charlie', 'https://api.dicebear.com/7.x/avataaars/svg?seed=charlie', 0),
(4, '1000000000000004', 'david', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'David', 'https://api.dicebear.com/7.x/avataaars/svg?seed=david', 0),
(5, '1000000000000005', 'eve', '$2a$10$N9qo8uLOic8dMIPhF0eveOKXyzv1W5GW.JpJgvzYsXnw7aEXQXnWe', 'Eve', 'https://api.dicebear.com/7.x/avataaars/svg?seed=eve', 0);

-- 重置用户序列
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));

-- 插入好友关系 (双向)
-- Alice(1) <-> Bob(2), Charlie(3), Eve(5)
-- Bob(2) <-> Charlie(3), David(4)
-- Charlie(3) <-> David(4)
INSERT INTO friends (id, object_code, user_id, friend_id, remark) VALUES
-- Alice 的好友
(1, '2000000000000001', 1, 2, 'Bob'),
(2, '2000000000000002', 2, 1, 'Alice'),
(3, '2000000000000003', 1, 3, 'Charlie'),
(4, '2000000000000004', 3, 1, 'Alice'),
(5, '2000000000000005', 1, 5, 'Eve'),
(6, '2000000000000006', 5, 1, 'Alice'),
-- Bob 的好友
(7, '2000000000000007', 2, 3, ''),
(8, '2000000000000008', 3, 2, ''),
(9, '2000000000000009', 2, 4, 'Dave'),
(10, '2000000000000010', 4, 2, 'Bob'),
-- Charlie 和 David
(11, '2000000000000011', 3, 4, ''),
(12, '2000000000000012', 4, 3, '');

-- 重置好友序列
SELECT setval('friends_id_seq', (SELECT MAX(id) FROM friends));
