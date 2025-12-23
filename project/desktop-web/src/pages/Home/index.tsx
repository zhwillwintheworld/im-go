import { Layout, Avatar, Tooltip } from 'antd';
import {
    MessageOutlined,
    PlayCircleOutlined,
    SettingOutlined,
    LogoutOutlined,
} from '@ant-design/icons';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import styles from './Home.module.css';

const { Sider, Content } = Layout;

type NavKey = 'chat' | 'game' | 'welcome';

function Home() {
    const navigate = useNavigate();
    const [activeNav, setActiveNav] = useState<NavKey>('welcome');
    const logout = useAuthStore((state) => state.logout);

    const handleNavClick = (key: NavKey) => {
        setActiveNav(key);
        if (key === 'chat') {
            navigate('/chat');
        } else if (key === 'game') {
            navigate('/game');
        }
    };

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    // 欢迎页内容
    const renderWelcome = () => (
        <div className={styles.welcomeContainer}>
            <div>
                <h1 className={styles.welcomeTitle}>欢迎回来 👋</h1>
                <p className={styles.welcomeSubtitle}>选择你想要进入的功能模块</p>
            </div>
            <div className={styles.cardContainer}>
                <div
                    className={styles.featureCard}
                    onClick={() => handleNavClick('chat')}
                >
                    <MessageOutlined className={styles.featureIcon} />
                    <span className={styles.featureTitle}>IM 聊天</span>
                    <span className={styles.featureDesc}>会话 · 好友 · 群组</span>
                </div>
                <div
                    className={styles.featureCard}
                    onClick={() => handleNavClick('game')}
                >
                    <PlayCircleOutlined className={styles.featureIcon} />
                    <span className={styles.featureTitle}>游戏中心</span>
                    <span className={styles.featureDesc}>开始游戏 · 查看战绩</span>
                </div>
            </div>
        </div>
    );

    return (
        <Layout className={styles.layout}>
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
                            className={`${styles.navItem} ${activeNav === 'chat' ? styles.active : ''}`}
                            onClick={() => handleNavClick('chat')}
                        >
                            <MessageOutlined className={styles.navIcon} />
                            <span className={styles.navLabel}>聊天</span>
                        </div>
                    </Tooltip>

                    <Tooltip title="游戏" placement="right">
                        <div
                            className={`${styles.navItem} ${activeNav === 'game' ? styles.active : ''}`}
                            onClick={() => handleNavClick('game')}
                        >
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

            <Content className={styles.mainContent}>
                {renderWelcome()}
            </Content>

            {/* 移动端底部导航 */}
            <div className={styles.mobileNav}>
                <div className={styles.mobileNavInner}>
                    <div
                        className={`${styles.mobileNavItem} ${activeNav === 'chat' ? styles.active : ''}`}
                        onClick={() => handleNavClick('chat')}
                    >
                        <MessageOutlined className={styles.mobileNavIcon} />
                        <span>聊天</span>
                    </div>
                    <div
                        className={`${styles.mobileNavItem} ${activeNav === 'game' ? styles.active : ''}`}
                        onClick={() => handleNavClick('game')}
                    >
                        <PlayCircleOutlined className={styles.mobileNavIcon} />
                        <span>游戏</span>
                    </div>
                    <div className={styles.mobileNavItem}>
                        <SettingOutlined className={styles.mobileNavIcon} />
                        <span>设置</span>
                    </div>
                    <div className={styles.mobileNavItem} onClick={handleLogout}>
                        <LogoutOutlined className={styles.mobileNavIcon} />
                        <span>退出</span>
                    </div>
                </div>
            </div>
        </Layout>
    );
}

export default Home;
