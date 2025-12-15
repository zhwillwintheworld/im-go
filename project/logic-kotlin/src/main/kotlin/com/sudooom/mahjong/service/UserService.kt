package com.sudooom.mahjong.service

import org.slf4j.LoggerFactory
import org.springframework.data.redis.core.ReactiveRedisTemplate
import org.springframework.stereotype.Service
import java.time.Duration

@Service
class UserService(
    private val reactiveRedisTemplate: ReactiveRedisTemplate<String, String>
) {
    private val logger = LoggerFactory.getLogger(javaClass)

    companion object {
        private const val USER_LOCATION_KEY_PREFIX = "im:user:location:"
        private val LOCATION_TTL = Duration.ofHours(24)
    }

    /**
     * 注册用户位置
     */
    suspend fun registerUserLocation(
        userId: Long,
        accessNodeId: String,
        connId: Long,
        deviceId: String,
        platform: String
    ) {
        val key = "$USER_LOCATION_KEY_PREFIX$userId"
        val field = "$accessNodeId:$connId"
        val value = """{"accessNodeId":"$accessNodeId","connId":$connId,"deviceId":"$deviceId","platform":"$platform"}"""

        reactiveRedisTemplate.opsForHash<String, String>()
            .put(key, field, value)
            .subscribe()

        reactiveRedisTemplate.expire(key, LOCATION_TTL).subscribe()

        logger.debug("Registered user location: userId=$userId, accessNodeId=$accessNodeId")
    }

    /**
     * 注销用户位置
     */
    suspend fun unregisterUserLocation(userId: Long, connId: Long, accessNodeId: String) {
        val key = "$USER_LOCATION_KEY_PREFIX$userId"
        val field = "$accessNodeId:$connId"

        reactiveRedisTemplate.opsForHash<String, String>()
            .remove(key, field)
            .subscribe()

        logger.debug("Unregistered user location: userId=$userId, connId=$connId")
    }
}
