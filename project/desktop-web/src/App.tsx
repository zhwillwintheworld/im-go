import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { Spin } from 'antd';

const Login = lazy(() => import('./pages/Login'));
const Chat = lazy(() => import('./pages/Chat'));

const Loading = () => (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
    </div>
);

function App() {
    return (
        <Suspense fallback={<Loading />}>
            <Routes>
                <Route path="/login" element={<Login />} />
                <Route path="/chat" element={<Chat />} />
                <Route path="*" element={<Navigate to="/login" replace />} />
            </Routes>
        </Suspense>
    );
}

export default App;
