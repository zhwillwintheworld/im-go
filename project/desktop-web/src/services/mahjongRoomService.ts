import * as flatbuffers from 'flatbuffers';
import { transportManager } from '@/services/transport/WebTransportManager';
import { IMProtocol } from '@/services/protocol/IMProtocol';
import { messageDispatcher, ResponsePayload } from '@/services/messageDispatcher';
import {
    RoomAction,
    RoomEvent,
    RoomReq,
    RoomPush,
    RoomInfo,
    GameType,
} from '@/protocol/im/protocol';

type RoomUpdateCallback = (roomInfo: RoomInfo) => void;

/**
 * 麻将房间消息服务
 * 处理房间相关的所有消息发送和接收
 */
class MahjongRoomService {
    private roomUpdateCallbacks: RoomUpdateCallback[] = [];

    constructor() {
        this.initMessageListeners();
    }

    /**
     * 初始化消息监听器
     */
    private initMessageListeners(): void {
        // 监听房间推送消息
        messageDispatcher.register(ResponsePayload.RoomPush, (payload, reqId) => {
            if (!payload) return;

            try {
                const buf = new flatbuffers.ByteBuffer(payload);
                const roomPush = RoomPush.getRootAsRoomPush(buf);
                const event = roomPush.event();
                const roomInfo = roomPush.roomInfo();

                console.log('[MahjongRoomService] Received room event:', RoomEvent[event], roomInfo);

                // 分发房间状态更新
                if (roomInfo) {
                    this.notifyRoomUpdate(roomInfo);
                }
            } catch (error) {
                console.error('[MahjongRoomService] Failed to parse room push:', error);
            }
        });
    }

    /**
     * 通知所有监听器房间状态更新
     */
    private notifyRoomUpdate(roomInfo: RoomInfo): void {
        for (const callback of this.roomUpdateCallbacks) {
            try {
                callback(roomInfo);
            } catch (error) {
                console.error('[MahjongRoomService] Room update callback error:', error);
            }
        }
    }

    /**
     * 发送房间请求
     */
    private async sendRoomRequest(
        action: RoomAction,
        roomId: string,
        targetSeatIndex: number = -1
    ): Promise<void> {
        const builder = new flatbuffers.Builder(256);

        // 构建房间ID
        const roomIdOffset = builder.createString(roomId);

        // 构建 RoomReq
        RoomReq.startRoomReq(builder);
        RoomReq.addAction(builder, action);
        RoomReq.addGameType(builder, GameType.MAHJONG);
        RoomReq.addRoomId(builder, roomIdOffset);

        // 注意：现有协议可能还没有 targetSeatIndex 字段
        // 如果使用，需要后端更新 schema
        // RoomReq.addTargetSeatIndex(builder, targetSeatIndex);

        const roomReqOffset = RoomReq.endRoomReq(builder);
        builder.finish(roomReqOffset);

        const payload = builder.asUint8Array();

        // 发送请求 - 使用通用的 request 方法
        const reqId = `mahjong-room-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
        const requestBuilder = new flatbuffers.Builder(512);
        const reqIdOffset = requestBuilder.createString(reqId);

        // 构建 ClientRequest
        const { ClientRequest, RequestPayload } = await import('@/protocol/im/protocol');
        const payloadOffset = ClientRequest.createPayloadVector(requestBuilder, payload);

        ClientRequest.startClientRequest(requestBuilder);
        ClientRequest.addReqId(requestBuilder, reqIdOffset);
        ClientRequest.addTimestamp(requestBuilder, BigInt(Date.now()));
        ClientRequest.addPayloadType(requestBuilder, RequestPayload.RoomReq);
        ClientRequest.addPayload(requestBuilder, payloadOffset);
        const clientReqOffset = ClientRequest.endClientRequest(requestBuilder);
        requestBuilder.finish(clientReqOffset);

        // 发送帧
        const { FrameType } = await import('@/services/protocol/IMProtocol');
        const frameBuilder = new flatbuffers.Builder(1024);
        const frameBody = requestBuilder.asUint8Array();
        const frame = new Uint8Array(5 + frameBody.length);
        const view = new DataView(frame.buffer);
        view.setUint32(0, frameBody.length, false);
        view.setUint8(4, FrameType.Request);
        frame.set(frameBody, 5);

        await transportManager.send(frame);

        console.log('[MahjongRoomService] Sent room request:', RoomAction[action], {
            roomId,
            targetSeatIndex,
        });
    }

    /**
     * 切换准备状态
     */
    async toggleReady(roomId: string): Promise<void> {
        await this.sendRoomRequest(RoomAction.READY, roomId);
    }

    /**
     * 换座位（占座）
     * @param roomId 房间ID
     * @param seatIndex 座位索引 (0=东, 1=南, 2=西, 3=北)
     */
    async takeSeat(roomId: string, seatIndex: number): Promise<void> {
        // 使用 JOIN 动作，后端可以根据 seatIndex 分配座位
        await this.sendRoomRequest(RoomAction.JOIN, roomId, seatIndex);
    }

    /**
     * 离开座位
     */
    async leaveSeat(roomId: string): Promise<void> {
        await this.sendRoomRequest(RoomAction.LEAVE, roomId);
    }

    /**
     * 开始游戏（房主）
     * 注意：需要后端支持 START_GAME 动作
     */
    async startGame(roomId: string): Promise<void> {
        // 如果后端已支持 START_GAME，使用它
        // 否则可能需要通过其他方式触发
        // await this.sendRoomRequest(RoomAction.START_GAME, roomId);

        // 临时方案：可能需要调用 REST API
        console.log('[MahjongRoomService] Start game request:', roomId);
        // TODO: 实现开始游戏逻辑
    }

    /**
     * 监听房间状态更新
     * @returns 取消监听的函数
     */
    onRoomUpdate(callback: RoomUpdateCallback): () => void {
        this.roomUpdateCallbacks.push(callback);

        // 返回取消监听函数
        return () => {
            const index = this.roomUpdateCallbacks.indexOf(callback);
            if (index >= 0) {
                this.roomUpdateCallbacks.splice(index, 1);
            }
        };
    }
}

// 导出单例
export const mahjongRoomService = new MahjongRoomService();
