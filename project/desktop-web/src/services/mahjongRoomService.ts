import * as flatbuffers from 'flatbuffers';
import { transportManager } from '@/services/transport/WebTransportManager';
import { FrameType } from '@/services/protocol/IMProtocol';
import { messageDispatcher, ResponsePayload } from '@/services/messageDispatcher';
import {
    RoomAction,
    RoomEvent,
    RoomReq,
    RoomPush,
    RoomInfo,
    GameType,
    ClientRequest,
    RequestPayload
} from '@/im/protocol';

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
        targetSeatIndex: number = -1,
        roomConfig: string = ''
    ): Promise<void> {
        const builder = new flatbuffers.Builder(256);

        // 构建房间ID和配置
        const roomIdOffset = builder.createString(roomId);
        const roomConfigOffset = roomConfig ? builder.createString(roomConfig) : 0;

        // 构建 RoomReq
        RoomReq.startRoomReq(builder);
        RoomReq.addAction(builder, action);
        RoomReq.addGameType(builder, GameType.HT_MAHJONG);
        RoomReq.addRoomId(builder, roomIdOffset);
        if (roomConfigOffset) {
            RoomReq.addRoomConfig(builder, roomConfigOffset);
        }

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
        const payloadOffset = ClientRequest.createPayloadVector(requestBuilder, payload);

        ClientRequest.startClientRequest(requestBuilder);
        ClientRequest.addReqId(requestBuilder, reqIdOffset);
        ClientRequest.addTimestamp(requestBuilder, BigInt(Date.now()));
        ClientRequest.addPayloadType(requestBuilder, RequestPayload.RoomReq);
        ClientRequest.addPayload(requestBuilder, payloadOffset);
        const clientReqOffset = ClientRequest.endClientRequest(requestBuilder);
        requestBuilder.finish(clientReqOffset);

        // 发送帧
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
            roomConfig,
        });
    }

    /**
     * 创建房间
     * @param roomConfig 房间配置 JSON 字符串
     * @returns Promise<string> 返回创建的房间 ID
     */
    async createRoom(roomConfig: string): Promise<void> {
        // 使用临时的房间 ID，实际的房间 ID 会在响应中返回
        await this.sendRoomRequest(RoomAction.CREATE, '', -1, roomConfig);
    }

    /**
     * 加入房间
     * @param roomId 房间 ID
     * @param password 房间密码（可选）
     * @param seatIndex 座位索引（可选，-1 表示自动分配）
     */
    async joinRoom(roomId: string, password?: string, seatIndex: number = -1): Promise<void> {
        // 如果有密码，可以放在 roomConfig 中
        const config = password ? JSON.stringify({ password }) : '';
        await this.sendRoomRequest(RoomAction.JOIN, roomId, seatIndex, config);
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
        // 使用 CHANGE_SEAT 动作
        await this.sendRoomRequest(RoomAction.CHANGE_SEAT, roomId, seatIndex);
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
        await this.sendRoomRequest(RoomAction.START_GAME, roomId);
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
