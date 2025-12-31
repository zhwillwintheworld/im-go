import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { message } from 'antd';
import { EyeOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons';
import { mahjongRoomService } from '@/services/mahjongRoomService';
import { useIMStore } from '@/stores/imStore';
import { RoomInfo, RoomPlayer as FBRoomPlayer } from '@/im/protocol';
import styles from './Room.module.css';

interface Player {
    id: string;
    name: string;
    avatar?: string;
    isReady: boolean;
    isOwner: boolean;
}

interface Spectator {
    id: string;
    name: string;
}

type SeatPosition = 'east' | 'south' | 'west' | 'north';

const POSITION_LABELS: Record<SeatPosition, string> = {
    east: '东家',
    south: '南家',
    west: '西家',
    north: '北家',
};

function MahjongRoom() {
    const navigate = useNavigate();
    const { roomId } = useParams<{ roomId: string }>();
    const imStatus = useIMStore((state) => state.status);

    // 模拟数据
    const [seats, setSeats] = useState<Record<SeatPosition, Player | null>>({
        east: { id: '1', name: '张三', isReady: true, isOwner: true },
        south: { id: '2', name: '李四', isReady: false, isOwner: false },
        west: null,
        north: null,
    });

    const [spectators] = useState<Spectator[]>([
        { id: '3', name: '王五' },
        { id: '4', name: '赵六' },
        { id: '5', name: '钱七' },
    ]);

    const [myId] = useState('1'); // 当前用户 ID
    const [isReady, setIsReady] = useState(false);
    const [chatInput, setChatInput] = useState('');

    const isOwner = seats.east?.id === myId && seats.east?.isOwner;
    const readyCount = Object.values(seats).filter(p => p?.isReady).length;
    const canStart = readyCount >= 4;

    // 监听房间状态更新
    useEffect(() => {
        if (!roomId || imStatus !== 'authenticated') return;

        const unsubscribe = mahjongRoomService.onRoomUpdate((roomInfo) => {
            const newSeats: Record<SeatPosition, Player | null> = {
                east: null, south: null, west: null, north: null,
            };
            const seatPositions: SeatPosition[] = ['east', 'south', 'west', 'north'];

            for (let i = 0; i < roomInfo.playersLength(); i++) {
                const fbPlayer = roomInfo.players(i);
                if (fbPlayer) {
                    const seatIndex = fbPlayer.seatIndex();
                    const position = seatPositions[seatIndex];
                    const user = fbPlayer.user();
                    if (position && user) {
                        newSeats[position] = {
                            id: fbPlayer.userId() || '',
                            name: user.nickname() || '未知',
                            isReady: fbPlayer.isReady(),
                            isOwner: fbPlayer.userId() === roomInfo.ownerId(),
                        };
                        if (fbPlayer.userId() === myId) {
                            setIsReady(fbPlayer.isReady());
                        }
                    }
                }
            }
            setSeats(newSeats);
        });

        return unsubscribe;
    }, [roomId, imStatus, myId]);

    const handleTakeSeat = async (position: SeatPosition) => {
        if (seats[position]) return;
        const seatPositions: SeatPosition[] = ['east', 'south', 'west', 'north'];
        const seatIndex = seatPositions.indexOf(position);
        try {
            await mahjongRoomService.takeSeat(roomId!, seatIndex);
            message.success(`正在占据 ${POSITION_LABELS[position]} 座位...`);
        } catch (error) {
            message.error('占座失败');
        }
    };

    const handleReady = async () => {
        try {
            await mahjongRoomService.toggleReady(roomId!);
            message.info(isReady ? '正在取消准备...' : '正在准备...');
        } catch (error) {
            message.error('操作失败');
        }
    };

    const handleStartGame = async () => {
        try {
            await mahjongRoomService.startGame(roomId!);
            message.info('正在开始游戏...');
            navigate(`/mahjong/game/${roomId}`);
        } catch (error) {
            message.error('开始游戏失败');
        }
    };

    const handleLeave = () => {
        navigate('/mahjong');
    };

    const handleSendChat = () => {
        if (!chatInput.trim()) return;
        message.info(`发送: ${chatInput}`);
        setChatInput('');
    };

    const renderSeat = (position: SeatPosition) => {
        const player = seats[position];

        if (!player) {
            return (
                <div
                    className={`${styles.seat} ${styles.empty}`}
                    onClick={() => handleTakeSeat(position)}
                >
                    <div className={styles.seatPosition}>{POSITION_LABELS[position]}</div>
                    <PlusOutlined className={styles.emptyIcon} />
                    <div className={styles.emptyText}>点击入座</div>
                </div>
            );
        }

        return (
            <div className={`${styles.seat} ${styles.occupied} ${player.isReady ? styles.ready : ''}`}>
                <div className={styles.seatPosition}>
                    {POSITION_LABELS[position]}
                    {player.isOwner && ' 👑'}
                </div>
                <div className={styles.seatAvatar}>
                    {player.name.charAt(0)}
                </div>
                <div className={styles.seatName}>{player.name}</div>
                <span className={`${styles.seatStatus} ${player.isReady ? styles.ready : styles.waiting}`}>
                    {player.isReady ? '✅ 已准备' : '⏳ 未准备'}
                </span>
            </div>
        );
    };

    return (
        <div className={styles.container}>
            {/* 顶部信息栏 */}
            <div className={styles.header}>
                <div className={styles.roomInfo}>
                    <span className={styles.roomId}>房间 #{roomId}</span>
                    <span className={styles.roomOwner}>房主: {seats.east?.name || '无'}</span>
                </div>
                <button className={styles.leaveBtn} onClick={handleLeave}>
                    退出房间
                </button>
            </div>

            {/* 主游戏区域 */}
            <div className={styles.mainArea}>
                {/* 麻将桌 */}
                <div className={styles.tableArea}>
                    {/* 北 */}
                    <div className={styles.seatRow}>
                        {renderSeat('north')}
                    </div>

                    {/* 西 + 桌子 + 东 */}
                    <div className={styles.seatMiddle}>
                        {renderSeat('west')}
                        <div className={styles.tableCenter}>
                            🀄 等待开始
                        </div>
                        {renderSeat('east')}
                    </div>

                    {/* 南 */}
                    <div className={styles.seatRow}>
                        {renderSeat('south')}
                    </div>
                </div>

                {/* 右侧边栏 */}
                <div className={styles.sidebar}>
                    {/* 观战列表 */}
                    <div className={styles.spectatorPanel}>
                        <div className={styles.panelTitle}>
                            <EyeOutlined /> 观战列表 ({spectators.length})
                        </div>
                        <div className={styles.spectatorList}>
                            {spectators.map(s => (
                                <div key={s.id} className={styles.spectator}>
                                    <EyeOutlined className={styles.spectatorIcon} />
                                    {s.name}
                                </div>
                            ))}
                        </div>
                    </div>

                    {/* 聊天区 */}
                    <div className={styles.chatPanel}>
                        <div className={styles.panelTitle}>💬 房间聊天</div>
                        <div className={styles.chatMessages}>
                            {/* 聊天消息列表占位 */}
                        </div>
                        <div className={styles.chatInput}>
                            <input
                                type="text"
                                placeholder="输入消息..."
                                value={chatInput}
                                onChange={(e) => setChatInput(e.target.value)}
                                onKeyDown={(e) => e.key === 'Enter' && handleSendChat()}
                            />
                            <button onClick={handleSendChat}>
                                <SendOutlined />
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            {/* 底部操作栏 */}
            <div className={styles.footer}>
                <button
                    className={`${styles.readyBtn} ${isReady ? styles.cancel : ''}`}
                    onClick={handleReady}
                >
                    {isReady ? '取消准备' : '🎮 准备'}
                </button>

                {isOwner && (
                    <button
                        className={styles.startBtn}
                        onClick={handleStartGame}
                    >
                        开始游戏 ({readyCount}/4)
                    </button>
                )}
            </div>
        </div>
    );
}

export default MahjongRoom;
