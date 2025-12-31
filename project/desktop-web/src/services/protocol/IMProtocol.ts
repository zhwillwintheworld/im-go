import * as flatbuffers from 'flatbuffers';
import {
    AuthRequest,
    ClientRequest,
    ClientResponse,
    HeartbeatReq,
    ChatSendReq,
    ConversationReadReq,
    Platform,
    RequestPayload,
    ResponsePayload,
    ChatType,
    MsgType
} from '@/im/protocol';

/**
 * 帧类型定义 - 与服务端 handler.go 保持一致
 */
export enum FrameType {
    Auth = 1,       // 认证请求 (AuthRequest)
    Request = 2,    // 普通请求 (ClientRequest)
    AuthAck = 3,    // 认证响应
    Response = 4,   // 普通响应 (ClientResponse)
}

/**
 * 帧头大小：4 bytes length + 1 byte frame type
 */
const FRAME_HEADER_SIZE = 5;

/**
 * 生成请求 ID
 */
function generateReqId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * IM 协议编解码器
 * 使用 FlatBuffers 进行序列化，符合 schema/message.fbs 设计
 */
export class IMProtocol {
    /**
     * 构建帧：添加帧头
     */
    private static buildFrame(frameType: FrameType, body: Uint8Array): Uint8Array {
        const frame = new Uint8Array(FRAME_HEADER_SIZE + body.length);
        const view = new DataView(frame.buffer);

        // 写入长度 (4 bytes, Big Endian)
        view.setUint32(0, body.length, false);
        // 写入帧类型 (1 byte)
        view.setUint8(4, frameType);
        // 写入 body
        frame.set(body, FRAME_HEADER_SIZE);

        return frame;
    }

    /**
     * 解析帧头
     */
    static parseFrameHeader(buffer: ArrayBuffer): { length: number; frameType: FrameType } | null {
        if (buffer.byteLength < FRAME_HEADER_SIZE) {
            return null;
        }
        const view = new DataView(buffer);
        return {
            length: view.getUint32(0, false),
            frameType: view.getUint8(4) as FrameType,
        };
    }

    /**
     * 提取帧 body
     */
    static extractBody(buffer: ArrayBuffer): Uint8Array {
        return new Uint8Array(buffer, FRAME_HEADER_SIZE);
    }

    // =========================================================================
    // 认证请求
    // =========================================================================

    /**
     * 创建认证请求帧
     */
    static createAuthRequest(token: string, deviceId: string, appVersion: string): Uint8Array {
        console.log('[IMProtocol] Creating AuthRequest:', {
            token: token.substring(0, 20) + '...',
            deviceId,
            platform: 'WEB',
            appVersion
        });

        const builder = new flatbuffers.Builder(256);

        const tokenOffset = builder.createString(token);
        const deviceIdOffset = builder.createString(deviceId);
        const appVersionOffset = builder.createString(appVersion);

        const authReqOffset = AuthRequest.createAuthRequest(
            builder,
            tokenOffset,
            deviceIdOffset,
            Platform.WEB,
            appVersionOffset
        );
        builder.finish(authReqOffset);

        return this.buildFrame(FrameType.Auth, builder.asUint8Array());
    }

    // =========================================================================
    // ClientRequest 包装的请求
    // =========================================================================

    /**
     * 创建心跳请求帧
     */
    static createHeartbeatRequest(): Uint8Array {
        // 1. 构建 HeartbeatReq payload
        const payloadBuilder = new flatbuffers.Builder(64);
        const heartbeatOffset = HeartbeatReq.createHeartbeatReq(
            payloadBuilder,
            BigInt(Date.now())
        );
        payloadBuilder.finish(heartbeatOffset);
        const payloadBytes = payloadBuilder.asUint8Array();

        // 2. 构建 ClientRequest
        const builder = new flatbuffers.Builder(256);
        const reqIdOffset = builder.createString(generateReqId());
        const payloadOffset = ClientRequest.createPayloadVector(builder, payloadBytes);

        const clientReqOffset = ClientRequest.createClientRequest(
            builder,
            reqIdOffset,
            BigInt(Date.now()),
            RequestPayload.HeartbeatReq,
            payloadOffset
        );
        builder.finish(clientReqOffset);

        return this.buildFrame(FrameType.Request, builder.asUint8Array());
    }

    /**
     * 创建聊天发送请求帧
     */
    static createChatSendRequest(
        chatType: ChatType,
        targetId: string,
        msgType: MsgType,
        content: string
    ): { frame: Uint8Array; reqId: string } {
        const reqId = generateReqId();

        // 1. 构建 ChatSendReq payload
        const payloadBuilder = new flatbuffers.Builder(512);
        const targetIdOffset = payloadBuilder.createString(targetId);
        const contentOffset = payloadBuilder.createString(content);

        ChatSendReq.startChatSendReq(payloadBuilder);
        ChatSendReq.addChatType(payloadBuilder, chatType);
        ChatSendReq.addTargetId(payloadBuilder, targetIdOffset);
        ChatSendReq.addMsgType(payloadBuilder, msgType);
        ChatSendReq.addContent(payloadBuilder, contentOffset);
        const chatReqOffset = ChatSendReq.endChatSendReq(payloadBuilder);
        payloadBuilder.finish(chatReqOffset);
        const payloadBytes = payloadBuilder.asUint8Array();

        // 2. 构建 ClientRequest
        const builder = new flatbuffers.Builder(1024);
        const reqIdOffset = builder.createString(reqId);
        const payloadOffset = ClientRequest.createPayloadVector(builder, payloadBytes);

        const clientReqOffset = ClientRequest.createClientRequest(
            builder,
            reqIdOffset,
            BigInt(Date.now()),
            RequestPayload.ChatSendReq,
            payloadOffset
        );
        builder.finish(clientReqOffset);

        return {
            frame: this.buildFrame(FrameType.Request, builder.asUint8Array()),
            reqId,
        };
    }

    /**
     * 创建会话已读请求帧
     * @param peerId 私聊对方ID（与 groupId 二选一）
     * @param groupId 群聊ID（与 peerId 二选一）
     * @param lastReadMsgId 最后已读消息ID
     */
    static createConversationReadRequest(
        peerId: string | null,
        groupId: string | null,
        lastReadMsgId: string
    ): { frame: Uint8Array; reqId: string } {
        const reqId = generateReqId();

        console.log('[IMProtocol] Creating ConversationReadRequest:', {
            reqId,
            peerId,
            groupId,
            lastReadMsgId
        });

        // 1. 构建 ConversationReadReq payload
        const payloadBuilder = new flatbuffers.Builder(256);
        const peerIdOffset = peerId ? payloadBuilder.createString(peerId) : 0;
        const groupIdOffset = groupId ? payloadBuilder.createString(groupId) : 0;
        const lastReadMsgIdOffset = payloadBuilder.createString(lastReadMsgId);

        ConversationReadReq.startConversationReadReq(payloadBuilder);
        if (peerIdOffset) ConversationReadReq.addPeerId(payloadBuilder, peerIdOffset);
        if (groupIdOffset) ConversationReadReq.addGroupId(payloadBuilder, groupIdOffset);
        ConversationReadReq.addLastReadMsgId(payloadBuilder, lastReadMsgIdOffset);
        const readReqOffset = ConversationReadReq.endConversationReadReq(payloadBuilder);
        payloadBuilder.finish(readReqOffset);
        const payloadBytes = payloadBuilder.asUint8Array();

        // 2. 构建 ClientRequest
        const builder = new flatbuffers.Builder(512);
        const reqIdOffset = builder.createString(reqId);
        const payloadOffset = ClientRequest.createPayloadVector(builder, payloadBytes);

        const clientReqOffset = ClientRequest.createClientRequest(
            builder,
            reqIdOffset,
            BigInt(Date.now()),
            RequestPayload.ConversationReadReq,
            payloadOffset
        );
        builder.finish(clientReqOffset);

        return {
            frame: this.buildFrame(FrameType.Request, builder.asUint8Array()),
            reqId,
        };
    }

    // =========================================================================
    // 响应解析
    // =========================================================================

    /**
     * 解析 ClientResponse
     */
    static parseClientResponse(body: Uint8Array): {
        reqId: string | null;
        code: number;
        msg: string | null;
        payloadType: ResponsePayload;
        payload: Uint8Array | null;
    } {
        const bb = new flatbuffers.ByteBuffer(body);
        const resp = ClientResponse.getRootAsClientResponse(bb);

        return {
            reqId: resp.reqId(),
            code: resp.code(),
            msg: resp.msg(),
            payloadType: resp.payloadType(),
            payload: resp.payloadArray(),
        };
    }
}

