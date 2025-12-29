import { IMProtocol, FrameType } from '../protocol/IMProtocol.js';
import { getUTC8TimeString } from '@/utils/time';

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
type MessageHandler = (frameType: FrameType, body: Uint8Array) => void;
type StatusHandler = (status: ConnectionStatus) => void;

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
    private sendStream: any | null = null;  // BidirectionalStream
    private sendWriter: WritableStreamDefaultWriter<Uint8Array> | null = null;
    private receiveReader: ReadableStreamDefaultReader<Uint8Array> | null = null;

    // 记录上次发送时间，用于分析空闲时间和延迟的关系
    private lastSendTime: number = 0;

    // 认证响应等待
    private authResolve: ((value: boolean) => void) | null = null;
    private authReject: ((reason?: any) => void) | null = null;

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

            // 检查是否在连接过程中被取消
            if (this.abortController.signal.aborted) {
                transport.close();
                this.transport = null;
                throw new Error('Connection aborted');
            }

            this.reconnectAttempts = 0;
            this.setStatus('connected');
            console.log('init stream')
            // 前置创建双向流和 writer/reader（避免首次发送延迟）
            await this.initializeStream();

            // 立即发送认证请求（连接建立后的首个请求）
            console.log('[WebTransport] Sending authentication request...');
            const authFrame = IMProtocol.createAuthRequest(
                authData.token,
                authData.deviceId,
                authData.appVersion
            );

            await this.send(authFrame);
            console.log('[WebTransport] Authentication request sent, waiting for response...');

            // 启动数据接收（在后台运行，不要 await，否则会阻塞）
            this.startReceiving();

            console.log('[WebTransport] ✅ Authentication successful');

            // 启动心跳（暂时关闭用于调试）
            // this.startHeartbeat();
        } catch (error) {
            console.error('[WebTransport] Connection failed:', error);
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

        // 关闭复用的发送流和 reader
        if (this.sendWriter) {
            this.sendWriter.close().catch(() => { });
            this.sendWriter = null;
        }
        if (this.receiveReader) {
            this.receiveReader.releaseLock();
            this.receiveReader = null;
        }
        this.sendStream = null;

        // 只有由于 'ready' promise 的特性，如果正在连接中调用 close() 会报错
        // 所以如果状态是 connecting，我们只 abort，让 connect 方法里的逻辑去关闭
        if (this.status === 'connected' && this.transport) {
            try {
                this.transport.close();
            } catch (e) {
                console.warn('[WebTransport] Error closing transport:', e);
            }
        }

        this.transport = null;
        this.isReceiving = false;
        this.setStatus('disconnected');
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
            console.log('[WebTransport] ✅ 前置创建双向流成功');
        } catch (error) {
            console.error('[WebTransport] 初始化流失败:', error);
            throw error;
        }
    }

    /**
     * 发送二进制数据（使用前置创建的流）
     */
    async send(data: Uint8Array): Promise<void> {
        if (!this.transport || this.status !== 'connected') {
            throw new Error('Not connected');
        }

        if (!this.sendWriter) {
            throw new Error('Send writer not initialized');
        }

        try {
            const startTime = performance.now();
            const startTimeString = getUTC8TimeString();

            // 计算距离上次发送的时间间隔
            const idleTime = this.lastSendTime > 0 ? startTime - this.lastSendTime : 0;
            const idleInfo = idleTime > 0 ? `, 空闲时间=${(idleTime / 1000).toFixed(2)}秒` : '';

            console.log(`[WebTransport] 📤 开始发送帧, 大小=${data.length}字节, 时间=${startTimeString}${idleInfo}`);

            // 直接写入，不等待 ready（减少延迟）
            await this.sendWriter.write(data);

            const endTime = performance.now();
            const endTimeString = getUTC8TimeString();
            const duration = endTime - startTime;
            this.lastSendTime = endTime;  // 记录本次发送时间

            console.log(`[WebTransport] ✅ 帧发送完成, 大小=${data.length}字节, 时间=${endTimeString}, 耗时=${duration.toFixed(2)}ms`);
        } catch (error) {
            console.error('[WebTransport] Send error:', error);
            // 出错时重置
            this.sendWriter = null;
            this.sendStream = null;
            throw error;
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

        console.log('[WebTransport] 开始接收数据...');

        try {
            await this.handleIncomingStream();
        } catch (error) {
            console.error('[WebTransport] Receive error:', error);
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
                const frameType = buffer[4];

                const totalFrameLength = 5 + bodyLength;

                // 确保缓冲区有完整的帧（帧头 + 帧体）
                while (buffer.length < totalFrameLength) {
                    const { done, value } = await this.receiveReader.read();
                    if (done) {
                        console.warn('[WebTransport] Stream ended before complete frame');
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
                if (frameType === 2 && this.authResolve) {  // FrameType.Response = 2
                    console.log('[WebTransport] Received Response frame, resolving auth');
                    this.authResolve(true);
                    this.authResolve = null;
                    this.authReject = null;
                }

                // 移除已处理的帧，保留剩余数据
                buffer = buffer.subarray(totalFrameLength);
            }
        } catch (error) {
            console.error('[WebTransport] Stream read error:', error);
        }
        // reader.releaseLock() 不需要，因为 reader 是成员变量，在 disconnect 时统一释放
    }

    private handleDisconnect(): void {
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
                console.debug('[WebTransport] Heartbeat sent');
            } catch (error) {
                console.error('[WebTransport] Heartbeat failed:', error);
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
