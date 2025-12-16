type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
type MessageHandler = (data: ArrayBuffer) => void;
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

    /**
     * 连接到 WebTransport 服务器
     * @param url WebTransport URL，格式: https://host:port
     */
    async connect(url: string): Promise<void> {
        this.url = url;
        this.setStatus('connecting');
        this.abortController = new AbortController();

        try {
            // 创建 WebTransport 连接
            this.transport = new WebTransport(url, {
                // 开发环境可能需要跳过证书验证（生产环境应移除）
                // serverCertificateHashes: [...],
            });

            // 等待连接就绪
            await this.transport.ready;

            this.setStatus('connected');
            this.reconnectAttempts = 0;
            this.startHeartbeat();

            // 启动数据接收
            this.startReceiving();
        } catch (error) {
            this.setStatus('disconnected');
            throw error;
        }
    }

    /**
     * 断开连接
     */
    disconnect(): void {
        this.stopHeartbeat();
        this.abortController?.abort();

        if (this.writer) {
            this.writer.close().catch(() => { });
            this.writer = null;
        }

        if (this.transport) {
            this.transport.close();
            this.transport = null;
        }

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
            // 使用单向流发送数据
            const stream = await this.transport.createUnidirectionalStream();
            const writer = stream.getWriter();
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
        if (!this.transport) return;

        try {
            // 接收服务器发起的单向流（用于推送消息）
            const reader = this.transport.incomingUnidirectionalStreams.getReader();

            while (true) {
                const { done, value: stream } = await reader.read();
                if (done) break;

                // 处理每个传入的流
                this.handleIncomingStream(stream);
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

            this.messageHandlers.forEach((handler) => handler(data.buffer));
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
        this.heartbeatInterval = window.setInterval(() => {
            // TODO: 发送心跳包
            // 可以通过 datagram 发送心跳
            if (this.transport?.datagrams) {
                // const writer = this.transport.datagrams.writable.getWriter();
                // await writer.write(heartbeatData);
                // writer.releaseLock();
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

export const transportManager = new WebTransportManager();
