package com.sudooom.mahjong.config

import io.nats.client.Connection
import io.nats.client.ConnectionListener
import io.nats.client.Nats
import io.nats.client.Options
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Value
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import java.time.Duration

@Configuration
class NatsConfig {

    private val logger = LoggerFactory.getLogger(javaClass)

    @Value("\${nats.server.url}")
    private lateinit var natsUrl: String

    @Value("\${nats.connection.max-reconnects}")
    private var maxReconnects: Int = -1

    @Value("\${nats.connection.reconnect-wait}")
    private lateinit var reconnectWait: Duration

    @Bean
    fun natsConnection(): Connection {
        val options = Options.Builder()
            .server(natsUrl)
            .maxReconnects(maxReconnects)
            .reconnectWait(reconnectWait)
            .connectionListener { conn, type ->
                when (type) {
                    ConnectionListener.Events.CONNECTED ->
                        logger.info("Connected to NATS: ${conn.serverInfo}")
                    ConnectionListener.Events.DISCONNECTED ->
                        logger.warn("Disconnected from NATS")
                    ConnectionListener.Events.RECONNECTED ->
                        logger.info("Reconnected to NATS: ${conn.serverInfo}")
                    else -> logger.debug("NATS event: $type")
                }
            }
            .build()

        return Nats.connect(options)
    }
}
