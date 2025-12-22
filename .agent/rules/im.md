---
trigger: always_on
---

1. 项目分为四个模块 access 持有用户长连接,logic 处理 im逻辑,web 作为 rest 服务,desktop-web 作为客户端与 im 系统通信
2. access-go 项目架构为 go + webtransport + flatbuffers 构建用户交互的服务器 持有用户连接，使用 redis 存储用户 location ，使用 nats 作为 broker 分发数据到 logic，也接收 logic 下发到 nats 的 message 再发往用户
3. logic-go 项目架构为 go + redis + postgressql, redis存储路由 热点数据，pg 存储消息
4. web-go 项目架构为 gin + redis + pg, redis 存储 token 信息,pg存储业务数据例如 好友 用户 战绩等等
5. desktop-web 项目架构为 react + tsx + typescript + vite + webtransport + antd
6. 我希望每个表都有create_at,update_at,deleted 字段 ,id 使用雪花id,create_at 与 update_at 是时间字段 对应创建时间与修改时间,deleted 表明逻辑删除字段 0 为正常 1 为已删除 ,
7. 修改了 schema.sql 就要修改对应的 model, 修改了 model 也要修改 schema.sql，新增 model 就要在schema 中新增表，新增表就要生成对应的 model
8. 绝对不允许在表中设置外键,可以在字段描述中表明这个字段对应某个表的主键，但绝对不允许使用外键
9. 建表语句中每个字段必须要有字段描述 字符串字段设置不可为空且需要加默认值为空字符串
11. 判断字符串字段是否为空不能使用 `column` = null 这个格式，需要使用`column` != ''
12. web-go 返回的response对象中id字段必须为string，因为前端js会丢精度