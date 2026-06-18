import { IMProtocol, FrameType } from '../protocol/IMProtocol.js';
import { logger } from '@/utils/logger';
import { latencyAnalyzer } from '@/services/WebTransportLatencyAnalyzer';

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
type MessageHandler = (frameType: FrameType, body: Uint8Array) => void;
type StatusHandler = (status: ConnectionStatus) => void;

const AUTH_TIMEOUT_MS = 5000;

/**
 * WebTransport 管理器
 * 用于通过 QUIC 协议与 Access Layer 进行双向通信
 */
class WebTransportManager {
    private transport: WebTransport | null = null;
    private url: string = '';
    private status: ConnectionStatus = 'disconnected';
    private reconnectAttempts: number = 0;
    private maxReconnectAttempts: number = 10;
    private reconnectDelay: number = 1000;
    private messageHandlers: MessageHandler[] = [];
    private statusHandlers: StatusHandler[] = [];
    private heartbeatInterval: number | null = null;
    private abortController: AbortController | null = null;
    private isReceiving: boolean = false;
    private authData: { token: string; deviceId: string; appVersion: string } | null = null;

    // 流复用优化：前置创建并复用双向流
    private sendStream: WebTransportBidirectionalStream | null = null;
    private sendWriter: WritableStreamDefaultWriter<Uint8Array> | null = null;
    private receiveReader: ReadableStreamDefaultReader<Uint8Array> | null = null;

    // 认证响应等待
    private authResolve: ((value: boolean) => void) | null = null;
    private authReject: ((reason?: unknown) => void) | null = null;
    private authTimeout: number | null = null;

    /**
     * 连接到 WebTransport 服务器并发送认证请求
     * @param url WebTransport URL，格式: https://host:port
     * @param authData 认证数据（token, deviceId, appVersion）
     */
    async connect(url: string, authData: { token: string; deviceId: string; appVersion: string }): Promise<void> {
        if (this.status === 'connected' || this.status === 'connecting') {
            return;
        }

        this.url = url;
        this.authData = authData;
        this.setStatus('connecting');
        this.abortController = new AbortController();

        try {
            latencyAnalyzer.recordConnectionStart();

            // 开发环境证书哈希 (14天有效期的自签名证书 wt-cert.pem)
            // WebTransport 要求自签名证书有效期不超过 14 天
            // 计算方法: openssl x509 -in wt-cert.pem -outform DER | openssl dgst -sha256 -binary | base64
            const certHash = 'dkaPP2DW0TTuleeuGambIGepaiBVNOQcKK1rFLdUr7I=';

            // 创建 WebTransport 连接
            const transport = new WebTransport(url, {
                congestionControl: "low-latency",
                allowPooling: true,
                requireUnreliable: false,
                serverCertificateHashes: [
                    { algorithm: "sha-256", value: new Uint8Array(atob(certHash).split("").map(c => c.charCodeAt(0))) }
                ],
            });

            this.transport = transport;

            // 等待连接就绪
            await transport.ready;
            const connectionLatency = latencyAnalyzer.recordConnectionReady();
            if (connectionLatency !== null) {
                logger.info(`[WebTransport] Connection ready latency=${connectionLatency.toFixed(2)}ms`);
            }

            // 检查是否在连接过程中被取消
            if (this.abortController.signal.aborted) {
                transport.close();
                this.transport = null;
                throw new Error('Connection aborted');
            }

            logger.debug('[WebTransport] Initializing stream');
            // 前置创建双向流和 writer/reader（避免首次发送延迟）
            await this.initializeStream();

            // 立即发送认证请求（连接建立后的首个请求）
            logger.info('[WebTransport] Sending authentication request...');
            const authFrame = IMProtocol.createAuthRequest(
                authData.token,
                authData.deviceId,
                authData.appVersion
            );

            const authResult = this.waitForAuthResponse();
            latencyAnalyzer.recordAuthStart();
            // 启动数据接收（在后台运行，不要 await，否则会阻塞）
            this.startReceiving();
            await this.writeFrame(authFrame);
            logger.info('[WebTransport] Authentication request sent, waiting for response...');
            await authResult;
            const authLatency = latencyAnalyzer.recordAuthSuccess();
            if (authLatency !== null) {
                logger.info(`[WebTransport] Auth latency=${authLatency.toFixed(2)}ms`);
            }

            if (this.abortController.signal.aborted) {
                throw new Error('Connection aborted');
            }

            this.reconnectAttempts = 0;
            this.setStatus('connected');
            logger.info('[WebTransport] Authentication successful');

            // 启动心跳（暂时关闭用于调试）
            this.startHeartbeat();
        } catch (error) {
            this.abortController?.abort();
            this.rejectAuth(error);
            this.cleanupTransport();
            logger.error('[WebTransport] Connection failed:', error);
            this.setStatus('disconnected');
            throw error;
        }
    }

    /**
     * 断开连接
     */
    disconnect(): void {
        this.stopHeartbeat();

        // 发出取消信号，通知 connect 过程停止
        this.abortController?.abort();
        this.rejectAuth(new Error('Connection aborted'));

        this.cleanupTransport();
        this.setStatus('disconnected');
    }

    private cleanupTransport(): void {
        if (this.sendWriter) {
            this.sendWriter.close().catch(() => { });
            this.sendWriter = null;
        }
        if (this.receiveReader) {
            this.receiveReader.releaseLock();
            this.receiveReader = null;
        }
        this.sendStream = null;

        if (this.transport) {
            try {
                this.transport.close();
            } catch (e) {
                logger.warn('[WebTransport] Error closing transport:', e);
            }
        }

        this.transport = null;
        this.isReceiving = false;
    }

    /**
     * 前置初始化双向流（在连接时调用）
     */
    private async initializeStream(): Promise<void> {
        if (!this.transport) {
            throw new Error('Transport not ready');
        }

        try {
            // 创建双向流用于发送和接收
            this.sendStream = await this.transport.createBidirectionalStream();
            this.sendWriter = this.sendStream.writable.getWriter();
            this.receiveReader = this.sendStream.readable.getReader();
            logger.info('[WebTransport] 前置创建双向流成功');
        } catch (error) {
            logger.error('[WebTransport] 初始化流失败:', error);
            throw error;
        }
    }

    /**
     * 发送二进制数据（使用前置创建的流）
     */
    async send(data: Uint8Array): Promise<void> {
        if (this.status !== 'connected') {
            throw new Error('Not connected');
        }

        await this.writeFrame(data);
    }

    private async writeFrame(data: Uint8Array): Promise<void> {
        if (!this.transport) {
            throw new Error('Transport not ready');
        }

        if (!this.sendWriter) {
            throw new Error('Send writer not initialized');
        }

        try {
            // 直接写入，不等待 ready（减少延迟）
            await this.sendWriter.write(data);
        } catch (error) {
            logger.error('[WebTransport] Send error:', error);
            // 出错时重置
            this.sendWriter = null;
            this.sendStream = null;
            throw error;
        }
    }

    private waitForAuthResponse(): Promise<boolean> {
        return new Promise((resolve, reject) => {
            this.authResolve = resolve;
            this.authReject = reject;
            this.authTimeout = window.setTimeout(() => {
                this.rejectAuth(new Error('Authentication timeout'));
            }, AUTH_TIMEOUT_MS);
        });
    }

    private resolveAuth(): void {
        this.clearAuthWaiter();
        this.authResolve?.(true);
        this.authResolve = null;
        this.authReject = null;
    }

    private rejectAuth(reason: unknown): void {
        this.clearAuthWaiter();
        this.authReject?.(reason);
        this.authResolve = null;
        this.authReject = null;
    }

    private clearAuthWaiter(): void {
        if (this.authTimeout) {
            clearTimeout(this.authTimeout);
            this.authTimeout = null;
        }
    }

    /**
     * 注册消息处理器
     */
    onMessage(handler: MessageHandler): void {
        this.messageHandlers.push(handler);
    }

    /**
     * 注册连接状态变化处理器
     */
    onStatusChange(handler: StatusHandler): void {
        this.statusHandlers.push(handler);
    }

    private setStatus(status: ConnectionStatus): void {
        this.status = status;
        this.statusHandlers.forEach((handler) => handler(status));
    }

    /**
     * 启动接收服务器推送的数据（使用前置创建的 reader）
     */
    private async startReceiving(): Promise<void> {
        if (!this.receiveReader || this.isReceiving) return;
        this.isReceiving = true;

        logger.info('[WebTransport] 开始接收数据...');

        try {
            await this.handleIncomingStream();
        } catch (error) {
            logger.error('[WebTransport] Receive error:', error);
            this.handleDisconnect();
        }
    }

    /**
     * 处理传入的流数据（使用前置创建的 reader 循环读取多帧）
     */
    private async handleIncomingStream(): Promise<void> {
        if (!this.receiveReader) {
            throw new Error('Receive reader not initialized');
        }

        try {
            let buffer = new Uint8Array(0);  // 缓冲区，存储未处理的数据

            while (true) {
                // 确保缓冲区至少有 5 字节（帧头）
                while (buffer.length < 5) {
                    const { done, value } = await this.receiveReader.read();
                    if (done) return;
                    if (value) {
                        const newBuffer = new Uint8Array(buffer.length + value.length);
                        newBuffer.set(buffer);
                        newBuffer.set(value, buffer.length);
                        buffer = newBuffer;
                    }
                }

                // 解析帧头：获取帧体长度（Big Endian）
                const dataView = new DataView(buffer.buffer, buffer.byteOffset, buffer.byteLength);
                const bodyLength = dataView.getUint32(0, false);  // false = Big Endian
                const frameType = buffer[4] as FrameType;

                const totalFrameLength = 5 + bodyLength;

                // 确保缓冲区有完整的帧（帧头 + 帧体）
                while (buffer.length < totalFrameLength) {
                    const { done, value } = await this.receiveReader.read();
                    if (done) {
                        logger.warn('[WebTransport] Stream ended before complete frame');
                        return;
                    }
                    if (value) {
                        const newBuffer = new Uint8Array(buffer.length + value.length);
                        newBuffer.set(buffer);
                        newBuffer.set(value, buffer.length);
                        buffer = newBuffer;
                    }
                }

                // 提取帧体
                const bodyData = buffer.subarray(5, totalFrameLength);

                // 分发消息
                this.messageHandlers.forEach((handler) => handler(frameType, bodyData));

                // 特殊处理：如果是Response帧，检查是否是认证响应
                if (frameType === FrameType.Response && this.authResolve) {
                    const resp = IMProtocol.parseClientResponse(bodyData);
                    if (resp.code === 0) {
                        this.resolveAuth();
                    } else {
                        this.rejectAuth(new Error(resp.msg ?? 'Authentication failed'));
                    }
                }

                // 移除已处理的帧，保留剩余数据
                buffer = buffer.subarray(totalFrameLength);
            }
        } catch (error) {
            logger.error('[WebTransport] Stream read error:', error);
        }
        // reader.releaseLock() 不需要，因为 reader 是成员变量，在 disconnect 时统一释放
    }

    private handleDisconnect(): void {
        if (this.abortController?.signal.aborted) {
            this.setStatus('disconnected');
            return;
        }

        if (this.reconnectAttempts < this.maxReconnectAttempts && this.authData) {
            this.setStatus('reconnecting');
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            setTimeout(() => {
                if (this.authData) {
                    this.connect(this.url, this.authData).catch(() => { });
                }
            }, delay);
        } else {
            this.setStatus('disconnected');
        }
    }

    private startHeartbeat(): void {
        this.heartbeatInterval = window.setInterval(async () => {
            if (this.status !== 'connected') return;

            try {
                const heartbeatFrame = IMProtocol.createHeartbeatRequest();
                await this.send(heartbeatFrame);
                logger.debug('[WebTransport] Heartbeat sent');
            } catch (error) {
                logger.error('[WebTransport] Heartbeat failed:', error);
            }
        }, 30000);
    }

    private stopHeartbeat(): void {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }
}

// 导出单例
export const transportManager = new WebTransportManager();
