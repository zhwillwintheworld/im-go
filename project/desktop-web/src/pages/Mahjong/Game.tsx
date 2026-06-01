import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import MahjongTile from '../../components/MahjongTile';
import styles from './Game.module.css';

// 麻将牌类型
type TileType = 'wan' | 'tiao' | 'tong';
type TileSuit = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9;

interface MahjongTile {
    id: string;
    type: TileType;
    value: TileSuit;
    selected?: boolean;
}

interface Player {
    id: string;
    name: string;
    handTilesCount: number;
    discardTiles: MahjongTile[];
    position: 'east' | 'south' | 'west' | 'north';
    isActive: boolean;
}

function MahjongGame() {
    const navigate = useNavigate();
    const { roomId } = useParams<{ roomId: string }>();

    const [myHandTiles, setMyHandTiles] = useState<MahjongTile[]>([]);
    const [selectedTileId, setSelectedTileId] = useState<string | null>(null);
    const [gameInfo] = useState({
        currentRound: 1,
        remainingTiles: 88,
        isDealerTurn: true,
    });

    const [players] = useState<Player[]>([
        { id: '1', name: '张三', handTilesCount: 13, discardTiles: [], position: 'east', isActive: true },
        { id: '2', name: '李四', handTilesCount: 13, discardTiles: [], position: 'south', isActive: false },
        { id: '3', name: '王五', handTilesCount: 13, discardTiles: [], position: 'west', isActive: false },
        { id: '4', name: '赵六', handTilesCount: 13, discardTiles: [], position: 'north', isActive: false },
    ]);

    // 初始化手牌
    useEffect(() => {
        const initialTiles: MahjongTile[] = [
            { id: '1', type: 'wan', value: 1 },
            { id: '2', type: 'wan', value: 2 },
            { id: '3', type: 'wan', value: 3 },
            { id: '4', type: 'wan', value: 4 },
            { id: '5', type: 'wan', value: 5 },
            { id: '6', type: 'tiao', value: 1 },
            { id: '7', type: 'tiao', value: 2 },
            { id: '8', type: 'tiao', value: 3 },
            { id: '9', type: 'tiao', value: 4 },
            { id: '10', type: 'tong', value: 1 },
            { id: '11', type: 'tong', value: 2 },
            { id: '12', type: 'tong', value: 3 },
            { id: '13', type: 'tong', value: 4 },
        ];
        setMyHandTiles(initialTiles);
    }, []);

    const handleTileClick = (tileId: string) => {
        setSelectedTileId(selectedTileId === tileId ? null : tileId);
    };

    const handleDiscard = () => {
        if (!selectedTileId) {
            message.warning('请选择要打出的牌');
            return;
        }

        const tileToDiscard = myHandTiles.find(t => t.id === selectedTileId);
        if (tileToDiscard) {
            const numberMap: Record<number, string> = {
                1: '一', 2: '二', 3: '三', 4: '四', 5: '五',
                6: '六', 7: '七', 8: '八', 9: '九',
            };
            const typeMap = { wan: '万', tiao: '条', tong: '筒' };
            message.info(`打出 ${numberMap[tileToDiscard.value]}${typeMap[tileToDiscard.type]}`);
            setMyHandTiles(prev => prev.filter(t => t.id !== selectedTileId));
            setSelectedTileId(null);
        }
    };

    const handleLeave = () => {
        navigate(`/mahjong/room/${roomId}`);
    };

    const renderOtherPlayer = (player: Player, position: 'top' | 'left' | 'right') => {
        return (
            <div className={`${styles.otherPlayer} ${styles[position]}`}>
                <div className={styles.playerInfo}>
                    <span className={styles.playerName}>
                        {player.name} {player.isActive && '⏰'}
                    </span>
                    <span className={styles.tileCount}>{player.handTilesCount} 张</span>
                </div>
                <div className={styles.handTiles}>
                    {Array.from({ length: player.handTilesCount }).map((_, i) => (
                        <div key={i} className={styles.hiddenTile}>🀫</div>
                    ))}
                </div>
                {player.discardTiles.length > 0 && (
                    <div className={styles.discardArea}>
                        {player.discardTiles.map(tile => (
                            <MahjongTile
                                key={tile.id}
                                type={tile.type}
                                value={tile.value}
                                size="small"
                            />
                        ))}
                    </div>
                )}
            </div>
        );
    };

    return (
        <div className={styles.container}>
            {/* 顶部信息栏 */}
            <div className={styles.header}>
                <button className={styles.backBtn} onClick={handleLeave}>
                    <ArrowLeftOutlined /> 返回房间
                </button>
                <div className={styles.gameInfo}>
                    <span className={styles.infoItem}>第 {gameInfo.currentRound} 局</span>
                    <span className={styles.infoItem}>剩余: {gameInfo.remainingTiles} 张</span>
                    <span className={styles.infoItem}>房间 #{roomId}</span>
                </div>
            </div>

            {/* 游戏主区 */}
            <div className={styles.gameArea}>
                {/* 北家(对家) */}
                {renderOtherPlayer(players.find(p => p.position === 'north')!, 'top')}

                {/* 中间区域 */}
                <div className={styles.middleArea}>
                    {/* 西家 */}
                    {renderOtherPlayer(players.find(p => p.position === 'west')!, 'left')}

                    {/* 中央牌池 */}
                    <div className={styles.centerTable}>
                        <div className={styles.tableContent}>
                            <div className={styles.dealerMark}>🀄</div>
                            <div className={styles.roundInfo}>东风 一局</div>
                        </div>
                    </div>

                    {/* 东家 */}
                    {renderOtherPlayer(players.find(p => p.position === 'east')!, 'right')}
                </div>

                {/* 南家(我) */}
                <div className={styles.myPlayer}>
                    <div className={styles.myHandTiles}>
                        {myHandTiles.map(tile => (
                            <MahjongTile
                                key={tile.id}
                                type={tile.type}
                                value={tile.value}
                                selected={selectedTileId === tile.id}
                                onClick={() => handleTileClick(tile.id)}
                            />
                        ))}
                    </div>

                    {/* 操作按钮 */}
                    <div className={styles.actionButtons}>
                        <button className={styles.actionBtn} onClick={handleDiscard} disabled={!selectedTileId}>
                            🎯 打牌
                        </button>
                        <button className={styles.actionBtn} disabled>
                            🀄 胡
                        </button>
                        <button className={styles.actionBtn} disabled>
                            🀫 杠
                        </button>
                        <button className={styles.actionBtn} disabled>
                            🀞 碰
                        </button>
                        <button className={styles.actionBtn} disabled>
                            🀆 吃
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}

export default MahjongGame;
