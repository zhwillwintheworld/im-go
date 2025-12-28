import { IMProtocol, FrameType } from '../protocol/IMProtocol.js';

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
type MessageHandler = (frameType: FrameType, body: Uint8Array) => void;
type StatusHandler = (status: ConnectionStatus) => void;

/**
 * WebTransport 管理器
 * 用于通过 QUIC 协议与 Access Layer 进行双向通信
 */
class WebTransportManager {
    private transport: WebTransport | null = null;
    private writer: WritableStreamDefaultWriter<Uint8Array> | null = null;
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

    /**
     * 连接到 WebTransport 服务器
     * @param url WebTransport URL，格式: https://host:port
     */
    async connect(url: string): Promise<void> {
        if (this.status === 'connected' || this.status === 'connecting') {
            return;
        }

        this.url = url;
        this.setStatus('connecting');
        this.abortController = new AbortController();

        try {
            // 开发环境证书哈希 (mkcert 生成的 localhost+2.pem)
            // 用于绕过 Chrome 对 WebTransport/QUIC 自签名证书的限制
            const certHash = 'Rl4sCNy7rq4eKHkxJj4n5qjaltEN8o9+5T3Iou/97yw=';

            // 创建 WebTransport 连接
            const transport = new WebTransport(url, {
                serverCertificateHashes: [
                    {
                        algorithm: 'sha-256',
                        value: Uint8Array.from(atob(certHash), c => c.charCodeAt(0))
                    }
                ]
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

            this.setStatus('connected');
            this.reconnectAttempts = 0;

            // 启动心跳
            this.startHeartbeat();

            // 启动数据接收
            await this.startReceiving();
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

        if (this.writer) {
            this.writer.close().catch(() => { });
            this.writer = null;
        }

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
     * 发送二进制数据
     */
    async send(data: Uint8Array): Promise<void> {
        if (!this.transport || this.status !== 'connected') {
            throw new Error('Not connected');
        }

        try {
            // 使用双向流发送数据 (Server 端的 AcceptStream 仅接受双向流)
            const stream = await this.transport.createBidirectionalStream();
            const writer = stream.writable.getWriter();
            await writer.write(data);
            await writer.close();
        } catch (error) {
            console.error('[WebTransport] Send error:', error);
            throw error;
        }
    }

    /**
     * 通过双向流发送数据（用于需要响应的请求）
     */
    async sendBidirectional(data: Uint8Array): Promise<ArrayBuffer> {
        if (!this.transport || this.status !== 'connected') {
            throw new Error('Not connected');
        }

        const stream = await this.transport.createBidirectionalStream();
        const writer = stream.writable.getWriter();
        await writer.write(data);
        await writer.close();

        // 读取响应
        const reader = stream.readable.getReader();
        const chunks: Uint8Array[] = [];

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            if (value) chunks.push(value);
        }

        // 合并所有 chunks
        const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;
        for (const chunk of chunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }

        return result.buffer;
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

    /**
     * 获取当前连接状态
     */
    getStatus(): ConnectionStatus {
        return this.status;
    }

    private setStatus(status: ConnectionStatus): void {
        this.status = status;
        this.statusHandlers.forEach((handler) => handler(status));
    }

    /**
     * 启动接收服务器推送的数据
     */
    private async startReceiving(): Promise<void> {
        if (!this.transport || this.isReceiving) return;
        this.isReceiving = true;

        try {
            // 接收服务器发起的流（改为双向流，不再使用单向流）
            const reader = this.transport.incomingBidirectionalStreams.getReader();

            while (true) {
                const { done, value: stream } = await reader.read();
                if (done) break;

                // 处理每个传入的流 (读取其 readable 部分)
                this.handleIncomingStream(stream.readable);
            }
        } catch (error) {
            console.error('[WebTransport] Receive error:', error);
            this.handleDisconnect();
        }
    }

    /**
     * 处理传入的流数据
     */
    private async handleIncomingStream(stream: ReadableStream<Uint8Array>): Promise<void> {
        const reader = stream.getReader();
        const chunks: Uint8Array[] = [];

        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                if (value) chunks.push(value);
            }

            // 合并并分发消息
            const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
            const data = new Uint8Array(totalLength);
            let offset = 0;
            for (const chunk of chunks) {
                data.set(chunk, offset);
                offset += chunk.length;
            }

            // 解析帧头并分发
            const header = IMProtocol.parseFrameHeader(data.buffer);
            if (header) {
                const body = IMProtocol.extractBody(data.buffer);
                this.messageHandlers.forEach((handler) => handler(header.frameType, body));
            }
        } catch (error) {
            console.error('[WebTransport] Stream read error:', error);
        }
    }

    private handleDisconnect(): void {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.setStatus('reconnecting');
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            setTimeout(() => {
                this.connect(this.url).catch(() => { });
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
