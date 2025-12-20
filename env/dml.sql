-- ============================================
-- IM 系统 PostgreSQL 测试数据
-- ============================================
-- 使用方式: 在 start-postgres.sh 中自动执行
-- 注意: id 使用雪花ID格式，由应用层生成
-- ============================================

-- 插入测试用户
-- 密码: 123456 (bcrypt hash)
INSERT INTO public.users (id, username, password_hash, nickname, avatar, status, create_at, update_at, deleted) VALUES (260591601036824576, 'zhanghua', '$2a$10$vyKNfgSV0FyEYGST3mN.lOsnzZq0bOXDgQW2EaAp5zdAkvfKXENMe', '张华', 'https://example.com/avatar.png', 0, '2025-12-20 02:17:59.290889 +00:00', '2025-12-20 02:26:36.419043 +00:00', 0);
INSERT INTO public.users (id, username, password_hash, nickname, avatar, status, create_at, update_at, deleted) VALUES (260707671999516672, 'xuxinyuan', '$2a$10$IKmKIOQPf.9iK1CAnUuCGObpebWtEYRHF/1aiRpfQ/yed8WAp/hP2', '许馨元', '', 0, '2025-12-20 09:59:14.888078 +00:00', '2025-12-20 09:59:14.888078 +00:00', 0);

-- 插入好友关系 (双向)
INSERT INTO public.friends (id, user_id, friend_id, remark, create_at, update_at, deleted) VALUES (260708772584886272, 260591601036824576, 260707671999516672, '', '2025-12-20 10:03:37.293028 +00:00', '2025-12-20 10:03:37.293028 +00:00', 0);
INSERT INTO public.friends (id, user_id, friend_id, remark, create_at, update_at, deleted) VALUES (260708772584886273, 260707671999516672, 260591601036824576, '', '2025-12-20 10:03:37.293028 +00:00', '2025-12-20 10:03:37.293028 +00:00', 0);

