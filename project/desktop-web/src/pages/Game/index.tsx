import { Layout, Avatar, Tooltip, message } from 'antd';
import {
    MessageOutlined,
    PlayCircleOutlined,
    SettingOutlined,
    LogoutOutlined,
    TrophyOutlined,
    ThunderboltOutlined,
    ArrowLeftOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import styles from './Game.module.css';

const { Sider, Content } = Layout;

function Game() {
    const navigate = useNavigate();
    const logout = useAuthStore((state) => state.logout);

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    const handleStartGame = () => {
        message.info('游戏功能开发中...');
    };

    const handleViewRecords = () => {
        message.info('战绩功能开发中...');
    };

    return (
        <Layout className={styles.container}>
            <Sider className={styles.navSider} width={70}>
                <Avatar
                    size={40}
                    className={styles.userAvatar}
                    style={{ backgroundColor: '#7c3aed' }}
                >
                    U
                </Avatar>

                <div className={styles.navMenu}>
                    <Tooltip title="聊天" placement="right">
                        <div
                            className={styles.navItem}
                            onClick={() => navigate('/chat')}
                        >
                            <MessageOutlined className={styles.navIcon} />
                            <span className={styles.navLabel}>聊天</span>
                        </div>
                    </Tooltip>

                    <Tooltip title="游戏" placement="right">
                        <div className={`${styles.navItem} ${styles.active}`}>
                            <PlayCircleOutlined className={styles.navIcon} />
                            <span className={styles.navLabel}>游戏</span>
                        </div>
                    </Tooltip>
                </div>

                <div className={styles.navBottom}>
                    <Tooltip title="设置" placement="right">
                        <div className={styles.navItem}>
                            <SettingOutlined className={styles.navIcon} />
                        </div>
                    </Tooltip>
                    <Tooltip title="退出登录" placement="right">
                        <div className={styles.navItem} onClick={handleLogout}>
                            <LogoutOutlined className={styles.navIcon} />
                        </div>
                    </Tooltip>
                </div>
            </Sider>

            <Content className={styles.content}>
                <div className={styles.backBtn} onClick={() => navigate('/home')}>
                    <ArrowLeftOutlined /> 返回主页
                </div>

                <h1 className={styles.title}>🎮 游戏中心</h1>
                <p className={styles.subtitle}>选择你想要的操作</p>

                <div className={styles.cardContainer}>
                    <div className={styles.gameCard} onClick={handleStartGame}>
                        <ThunderboltOutlined className={styles.cardIcon} />
                        <span className={styles.cardTitle}>开始游戏</span>
                        <span className={styles.cardDesc}>匹配对手开始对战</span>
                    </div>

                    <div className={styles.gameCard} onClick={handleViewRecords}>
                        <TrophyOutlined className={styles.cardIcon} />
                        <span className={styles.cardTitle}>查看战绩</span>
                        <span className={styles.cardDesc}>历史对战记录</span>
                    </div>
                </div>
            </Content>
        </Layout>
    );
}

export default Game;
