type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
type MessageHandler = (data: ArrayBuffer) => void;
type StatusHandler = (status: ConnectionStatus) => void;

class WebSocketManager {
    private ws: WebSocket | null = null;
    private url: string = '';
    private status: ConnectionStatus = 'disconnected';
    private reconnectAttempts: number = 0;
    private maxReconnectAttempts: number = 10;
    private reconnectDelay: number = 1000;
    private messageHandlers: MessageHandler[] = [];
    private statusHandlers: StatusHandler[] = [];
    private heartbeatInterval: number | null = null;

    connect(url: string): Promise<void> {
        this.url = url;
        this.setStatus('connecting');

        return new Promise((resolve, reject) => {
            this.ws = new WebSocket(url);
            this.ws.binaryType = 'arraybuffer';

            this.ws.onopen = () => {
                this.setStatus('connected');
                this.reconnectAttempts = 0;
                this.startHeartbeat();
                resolve();
            };

            this.ws.onclose = () => {
                this.stopHeartbeat();
                this.handleDisconnect();
            };

            this.ws.onerror = (error) => {
                if (this.status === 'connecting') {
                    reject(error);
                }
            };

            this.ws.onmessage = (event) => {
                this.messageHandlers.forEach((handler) => handler(event.data));
            };
        });
    }

    disconnect(): void {
        this.stopHeartbeat();
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.setStatus('disconnected');
    }

    send(data: Uint8Array): void {
        if (this.ws && this.status === 'connected') {
            this.ws.send(data);
        }
    }

    onMessage(handler: MessageHandler): void {
        this.messageHandlers.push(handler);
    }

    onStatusChange(handler: StatusHandler): void {
        this.statusHandlers.push(handler);
    }

    private setStatus(status: ConnectionStatus): void {
        this.status = status;
        this.statusHandlers.forEach((handler) => handler(status));
    }

    private handleDisconnect(): void {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.setStatus('reconnecting');
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            setTimeout(() => {
                this.connect(this.url).catch(() => {});
            }, delay);
        } else {
            this.setStatus('disconnected');
        }
    }

    private startHeartbeat(): void {
        this.heartbeatInterval = window.setInterval(() => {
            // TODO: 发送心跳包
        }, 30000);
    }

    private stopHeartbeat(): void {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }
}

export const wsManager = new WebSocketManager();
