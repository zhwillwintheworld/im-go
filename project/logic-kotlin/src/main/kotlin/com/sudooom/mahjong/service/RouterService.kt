package com.sudooom.mahjong.service

import com.sudooom.mahjong.nats.MessagePublisher
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.reactive.asFlow
import org.slf4j.LoggerFactory
import org.springframework.data.redis.core.ReactiveRedisTemplate
import org.springframework.stereotype.Service

@Service
class RouterService(
    private val reactiveRedisTemplate: ReactiveRedisTemplate<String, String>,
    private val messagePublisher: MessagePublisher
) {
    private val logger = LoggerFactory.getLogger(javaClass)

    companion object {
        private const val USER_LOCATION_KEY_PREFIX = "im:user:location:"
    }

    /**
     * 获取用户所在的 Access 节点
     */
    suspend fun getUserLocations(userId: Long): List<UserLocation> {
        val key = "$USER_LOCATION_KEY_PREFIX$userId"
        return reactiveRedisTemplate.opsForHash<String, String>()
            .entries(key)
            .mapNotNull { entry ->
                try {
                    parseUserLocation(entry.value)
                } catch (e: Exception) {
                    null
                }
            }
            .asFlow()
            .toList()
    }

    /**
     * 路由消息到用户
     */
    suspend fun routeMessage(userId: Long, message: ByteArray) {
        val locations = getUserLocations(userId)

        if (locations.isEmpty()) {
            logger.debug("User $userId is offline, saving to offline storage")
            // TODO: 保存离线消息
            return
        }

        // 推送到所有在线设备
        locations.groupBy { it.accessNodeId }.forEach { (accessNodeId, _) ->
            messagePublisher.publishToAccess(accessNodeId, message)
        }
    }

    private fun parseUserLocation(json: String): UserLocation {
        // TODO: 使用 Jackson 解析
        return UserLocation(
            accessNodeId = "",
            connId = 0,
            deviceId = "",
            platform = ""
        )
    }
}

data class UserLocation(
    val accessNodeId: String,
    val connId: Long,
    val deviceId: String,
    val platform: String
)
