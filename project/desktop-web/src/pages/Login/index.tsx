import { Button, Form, Input, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import styles from './Login.module.css';

interface LoginForm {
    username: string;
    password: string;
}

function Login() {
    const navigate = useNavigate();
    const login = useAuthStore((state) => state.login);

    const onFinish = async (values: LoginForm) => {
        try {
            await login(values.username, values.password);
            message.success('登录成功');
            navigate('/chat');
        } catch (error) {
            message.error('登录失败，请检查用户名和密码');
        }
    };

    return (
        <div className={styles.container}>
            <Card className={styles.card} title="IM 登录">
                <Form
                    name="login"
                    onFinish={onFinish}
                    autoComplete="off"
                    size="large"
                >
                    <Form.Item
                        name="username"
                        rules={[{ required: true, message: '请输入用户名' }]}
                    >
                        <Input prefix={<UserOutlined />} placeholder="用户名" />
                    </Form.Item>

                    <Form.Item
                        name="password"
                        rules={[{ required: true, message: '请输入密码' }]}
                    >
                        <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                    </Form.Item>

                    <Form.Item>
                        <Button type="primary" htmlType="submit" block>
                            登录
                        </Button>
                    </Form.Item>
                </Form>
            </Card>
        </div>
    );
}

export default Login;
