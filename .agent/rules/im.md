---
trigger: always_on
---

1. 项目分为三个模块 access 持有用户连接,logic 处理逻辑,desktop-web 作为客户端与 im 系统通信
2. access 项目架构为 go + webtransport + flatbuffers 构建用户交互的服务器 持有用户连接，使用 redis 存储用户 location ，使用 nats 作为 broker 分发数据到 logic，也接收 logic 下发到 nats 的 message 再发往用户
3. logic 项目架构为 go + redis + postgressql , redis存储路由 热点数据，pg 存储消息
4. desktop-web 项目架构为 react + tsx + typescript + vite + webtransport + antd