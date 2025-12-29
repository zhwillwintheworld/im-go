import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol, FrameType } from '@/services/protocol/IMProtocol';
import { ResponsePayload } from '@/protocol/im/protocol/response-payload';
import { getUTC8TimeString } from '@/utils/time';

type MessageHandler = (payload: Uint8Array | null, reqId: string | null) => void;

/**
 * 消息分发器
 * 根据消息类型分发到不同的处理器
 */
class MessageDispatcher {
    private handlers: Map<ResponsePayload, MessageHandler[]> = new Map();
    private initialized = false;

    /**
     * 注册消息处理器
     */
    register(payloadType: ResponsePayload, handler: MessageHandler): void {
        const handlers = this.handlers.get(payloadType) || [];
        handlers.push(handler);
        this.handlers.set(payloadType, handlers);
    }

    /**
     * 移除消息处理器
     */
    unregister(payloadType: ResponsePayload, handler: MessageHandler): void {
        const handlers = this.handlers.get(payloadType) || [];
        const index = handlers.indexOf(handler);
        if (index >= 0) {
            handlers.splice(index, 1);
            this.handlers.set(payloadType, handlers);
        }
    }

    /**
     * 初始化监听器
     */
    init(): void {
        if (this.initialized) return;
        this.initialized = true;

        transportManager.onMessage((frameType: FrameType, body: Uint8Array) => {


            if (frameType === FrameType.Response) {
                this.handleResponse(body);
            } else if (frameType === FrameType.AuthAck) {
                console.log('[MessageDispatcher] AuthAck received');
                // 认证响应可以在这里处理，或者让 imStore 处理
            }
        });

        console.log('[MessageDispatcher] Initialized');
    }

    private handleResponse(body: Uint8Array): void {
        const resp = IMProtocol.parseClientResponse(body);
        console.log('[MessageDispatcher] ClientResponse:', {
            reqId: resp.reqId,
            code: resp.code,
            msg: resp.msg,
            payloadType: ResponsePayload[resp.payloadType],
        });

        // 分发到对应的处理器
        const handlers = this.handlers.get(resp.payloadType) || [];
        for (const handler of handlers) {
            try {
                handler(resp.payload, resp.reqId);
            } catch (e) {
                console.error('[MessageDispatcher] Handler error:', e);
            }
        }

        // 如果没有注册处理器，打印警告
        if (handlers.length === 0) {
            console.warn('[MessageDispatcher] No handler for payload type:', ResponsePayload[resp.payloadType]);
        }
    }
}

// 导出单例
export const messageDispatcher = new MessageDispatcher();

// 导出类型供其他模块使用
export { ResponsePayload };
