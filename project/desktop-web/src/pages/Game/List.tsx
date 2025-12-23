import { useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined } from '@ant-design/icons';
import styles from './List.module.css';

function GameList() {
    const navigate = useNavigate();

    return (
        <div className={styles.container}>
            <div className={styles.backBtn} onClick={() => navigate('/game')}>
                <ArrowLeftOutlined /> 返回游戏中心
            </div>

            <div className={styles.content}>
                <h1 className={styles.title}>🎮 选择游戏</h1>
                <p className={styles.subtitle}>选择一个游戏开始对战</p>

                <div className={styles.gameGrid}>
                    {/* 会同麻将 */}
                    <div
                        className={`${styles.gameCard} ${styles.mahjongCard}`}
                        onClick={() => navigate('/mahjong')}
                    >
                        <span className={styles.gameIcon}>🀄</span>
                        <span className={styles.gameName}>会同麻将</span>
                        <span className={styles.gameDesc}>经典四人麻将，创建或加入房间</span>
                    </div>

                    {/* 更多游戏占位 - 敬请期待 */}
                    <div className={`${styles.gameCard} ${styles.comingSoon}`}>
                        <span className={styles.gameIcon}>🎯</span>
                        <span className={styles.gameName}>更多游戏</span>
                        <span className={styles.gameDesc}>敬请期待</span>
                        <span className={styles.badge}>即将推出</span>
                    </div>
                </div>
            </div>
        </div>
    );
}

export default GameList;
