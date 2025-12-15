package com.sudooom.mahjong.service

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service

@Service
class MessageService {

    private val logger = LoggerFactory.getLogger(javaClass)

    /**
     * 保存消息到数据库
     */
    suspend fun saveMessage(message: Any): String {
        // TODO: 实现消息存储
        logger.debug("Saving message")
        return generateMessageId()
    }

    private fun generateMessageId(): String {
        return java.util.UUID.randomUUID().toString()
    }
}
