package com.sudooom.mahjong.nats

import io.nats.client.Connection
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component

@Component
class MessagePublisher(
    private val natsConnection: Connection
) {
    private val logger = LoggerFactory.getLogger(javaClass)

    /**
     * 推送消息到指定 Access 节点
     */
    fun publishToAccess(accessNodeId: String, message: ByteArray) {
        val subject = "im.access.$accessNodeId.downstream"
        natsConnection.publish(subject, message)
    }

    /**
     * 广播消息到所有 Access 节点
     */
    fun broadcast(message: ByteArray) {
        natsConnection.publish("im.access.broadcast", message)
    }
}
