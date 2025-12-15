package com.sudooom.mahjong.nats

import com.sudooom.mahjong.service.MessageService
import com.sudooom.mahjong.service.RouterService
import com.sudooom.mahjong.service.UserService
import io.nats.client.Connection
import io.nats.client.Dispatcher
import jakarta.annotation.PostConstruct
import jakarta.annotation.PreDestroy
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component

@Component
class MessageSubscriber(
    private val natsConnection: Connection,
    private val messageService: MessageService,
    private val userService: UserService,
    private val routerService: RouterService
) {
    private val logger = LoggerFactory.getLogger(javaClass)
    private val dispatcher: Dispatcher = natsConnection.createDispatcher()

    @PostConstruct
    fun start() {
        // 订阅上行消息 - 使用队列组实现负载均衡
        dispatcher.subscribe("im.logic.upstream", "logic-group") { msg ->
            CoroutineScope(Dispatchers.IO).launch {
                handleUpstreamMessage(msg.data)
            }
        }
        logger.info("NATS subscriber started, listening on im.logic.upstream")
    }

    private suspend fun handleUpstreamMessage(data: ByteArray) {
        // TODO: 解析 Protobuf 消息并分发处理
        logger.debug("Received upstream message, size: ${data.size}")
    }

    @PreDestroy
    fun stop() {
        dispatcher.unsubscribe("im.logic.upstream")
        logger.info("NATS subscriber stopped")
    }
}
