import { Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { Spin } from 'antd';
import { IMProvider } from './components/IMProvider';

const Login = lazy(() => import('./pages/Login'));
const Home = lazy(() => import('./pages/Home'));
const Chat = lazy(() => import('./pages/Chat'));
const Game = lazy(() => import('./pages/Game'));
const GameList = lazy(() => import('./pages/Game/List'));
const Mahjong = lazy(() => import('./pages/Mahjong'));
const MahjongRoom = lazy(() => import('./pages/Mahjong/Room'));
const MahjongGame = lazy(() => import('./pages/Mahjong/Game'));

const Loading = () => (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" />
    </div>
);

function App() {
    return (
        <IMProvider>
            <Suspense fallback={<Loading />}>
                <Routes>
                    <Route path="/login" element={<Login />} />
                    <Route path="/home" element={<Home />} />
                    <Route path="/chat" element={<Chat />} />
                    <Route path="/game" element={<Game />} />
                    <Route path="/game/list" element={<GameList />} />
                    <Route path="/mahjong" element={<Mahjong />} />
                    <Route path="/mahjong/room/:roomId" element={<MahjongRoom />} />
                    <Route path="/mahjong/game/:roomId" element={<MahjongGame />} />
                    <Route path="*" element={<Navigate to="/login" replace />} />
                </Routes>
            </Suspense>
        </IMProvider>
    );
}

export default App;
