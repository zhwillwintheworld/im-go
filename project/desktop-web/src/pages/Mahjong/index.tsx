import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Switch, message } from 'antd';
import { ArrowLeftOutlined, PlusOutlined, LoginOutlined } from '@ant-design/icons';
import styles from './Mahjong.module.css';

type ModalType = 'none' | 'create' | 'join';

interface RoomSettings {
    roomName: string;
    password: string;
    maxPlayers: number;
    allowSpectators: boolean;
    autoStart: boolean;
}

function Mahjong() {
    const navigate = useNavigate();
    const [modalType, setModalType] = useState<ModalType>('none');

    // 加入房间表单
    const [joinRoomId, setJoinRoomId] = useState('');
    const [joinPassword, setJoinPassword] = useState('');

    // 创建房间设置
    const [roomSettings, setRoomSettings] = useState<RoomSettings>({
        roomName: '',
        password: '',
        maxPlayers: 8,
        allowSpectators: true,
        autoStart: false,
    });

    const handleJoinRoom = () => {
        if (!joinRoomId.trim()) {
            message.error('请输入房间号');
            return;
        }
        // TODO: 调用 API 加入房间
        message.info(`加入房间 ${joinRoomId}...`);
        navigate(`/mahjong/room/${joinRoomId}`);
    };

    const handleCreateRoom = () => {
        if (!roomSettings.roomName.trim()) {
            message.error('请输入房间名称');
            return;
        }
        // TODO: 调用 API 创建房间
        message.info('创建房间中...');
        // 模拟创建成功后跳转
        navigate('/mahjong/room/123456');
    };

    const closeModal = () => {
        setModalType('none');
        setJoinRoomId('');
        setJoinPassword('');
    };

    // 加入房间弹窗
    const renderJoinModal = () => (
        <div className={styles.modal} onClick={closeModal}>
            <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
                <h2 className={styles.modalTitle}>🚪 加入房间</h2>

                <div className={styles.formGroup}>
                    <label className={styles.formLabel}>房间号</label>
                    <input
                        type="text"
                        className={styles.formInput}
                        placeholder="请输入6位房间号"
                        value={joinRoomId}
                        onChange={(e) => setJoinRoomId(e.target.value)}
                        maxLength={6}
                    />
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.formLabel}>密码</label>
                    <input
                        type="password"
                        className={styles.formInput}
                        placeholder="请输入房间密码（可选）"
                        value={joinPassword}
                        onChange={(e) => setJoinPassword(e.target.value)}
                    />
                    <span className={styles.formHint}>如果房间有密码才需要填写</span>
                </div>

                <div className={styles.modalActions}>
                    <button className={styles.cancelBtn} onClick={closeModal}>
                        取消
                    </button>
                    <button
                        className={`${styles.submitBtn} ${styles.cyan}`}
                        onClick={handleJoinRoom}
                    >
                        加入
                    </button>
                </div>
            </div>
        </div>
    );

    // 创建房间弹窗
    const renderCreateModal = () => (
        <div className={styles.modal} onClick={closeModal}>
            <div className={styles.modalContent} onClick={(e) => e.stopPropagation()}>
                <h2 className={styles.modalTitle}>➕ 开启房间</h2>

                <div className={styles.formGroup}>
                    <label className={styles.formLabel}>房间名称</label>
                    <input
                        type="text"
                        className={styles.formInput}
                        placeholder="给房间取个名字"
                        value={roomSettings.roomName}
                        onChange={(e) =>
                            setRoomSettings({ ...roomSettings, roomName: e.target.value })
                        }
                    />
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.formLabel}>房间密码（可选）</label>
                    <input
                        type="password"
                        className={styles.formInput}
                        placeholder="不填则为公开房间"
                        value={roomSettings.password}
                        onChange={(e) =>
                            setRoomSettings({ ...roomSettings, password: e.target.value })
                        }
                    />
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.formLabel}>最大人数</label>
                    <select
                        className={styles.formInput}
                        value={roomSettings.maxPlayers}
                        onChange={(e) =>
                            setRoomSettings({ ...roomSettings, maxPlayers: Number(e.target.value) })
                        }
                    >
                        <option value={4}>4 人</option>
                        <option value={6}>6 人</option>
                        <option value={8}>8 人</option>
                    </select>
                </div>

                <div className={styles.settingRow}>
                    <span className={styles.settingLabel}>允许观战</span>
                    <Switch
                        checked={roomSettings.allowSpectators}
                        onChange={(checked) =>
                            setRoomSettings({ ...roomSettings, allowSpectators: checked })
                        }
                    />
                </div>

                <div className={styles.settingRow}>
                    <span className={styles.settingLabel}>4人准备后自动开始</span>
                    <Switch
                        checked={roomSettings.autoStart}
                        onChange={(checked) =>
                            setRoomSettings({ ...roomSettings, autoStart: checked })
                        }
                    />
                </div>

                <div className={styles.modalActions}>
                    <button className={styles.cancelBtn} onClick={closeModal}>
                        取消
                    </button>
                    <button className={styles.submitBtn} onClick={handleCreateRoom}>
                        创建房间
                    </button>
                </div>
            </div>
        </div>
    );

    return (
        <div className={styles.container}>
            <div className={styles.backBtn} onClick={() => navigate('/game')}>
                <ArrowLeftOutlined /> 返回游戏中心
            </div>

            <div className={styles.content}>
                <h1 className={styles.title}>🀄 会同麻将</h1>
                <p className={styles.subtitle}>选择你的游戏方式</p>

                <div className={styles.optionContainer}>
                    <button
                        className={`${styles.optionBtn} ${styles.createBtn}`}
                        onClick={() => setModalType('create')}
                    >
                        <PlusOutlined className={styles.optionIcon} />
                        开启房间
                    </button>

                    <button
                        className={`${styles.optionBtn} ${styles.joinBtn}`}
                        onClick={() => setModalType('join')}
                    >
                        <LoginOutlined className={styles.optionIcon} />
                        加入房间
                    </button>
                </div>
            </div>

            {modalType === 'join' && renderJoinModal()}
            {modalType === 'create' && renderCreateModal()}
        </div>
    );
}

export default Mahjong;
