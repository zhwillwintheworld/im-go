import { useState, useEffect } from 'react';
import { Card, Statistic, Row, Col, Badge, Button } from 'antd';
import { ThunderboltOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { latencyAnalyzer, LatencyStats } from '@/services/WebTransportLatencyAnalyzer';

/**
 * 延迟监控组件
 * 实时显示 WebTransport 延迟统计信息
 */
export function LatencyMonitor() {
    const [stats, setStats] = useState<LatencyStats | null>(null);
    const [connectionStats, setConnectionStats] = useState<LatencyStats | null>(null);
    const [authStats, setAuthStats] = useState<LatencyStats | null>(null);
    const [pendingCount, setPendingCount] = useState(0);

    useEffect(() => {
        // 每秒更新一次统计数据
        const interval = setInterval(() => {
            setStats(latencyAnalyzer.getStats());
            setConnectionStats(latencyAnalyzer.getConnectionStats());
            setAuthStats(latencyAnalyzer.getAuthStats());
            setPendingCount(latencyAnalyzer.getPendingCount());
        }, 1000);

        return () => clearInterval(interval);
    }, []);

    const handlePrintReport = () => {
        latencyAnalyzer.printReport();
    };

    const handleReset = () => {
        latencyAnalyzer.reset();
        setStats(null);
        setConnectionStats(null);
        setAuthStats(null);
        setPendingCount(0);
    };

    if (!stats && !connectionStats && !authStats) {
        return (
            <Card
                title="📊 延迟监控"
                size="small"
                extra={<Badge status="default" text="无数据" />}
            >
                <p>发送消息后即可查看延迟统计</p>
            </Card>
        );
    }

    const getLatencyStatus = (avg: number) => {
        if (avg < 20) return 'success';
        if (avg < 50) return 'processing';
        if (avg < 100) return 'warning';
        return 'error';
    };

    const badgeAvg = stats?.avg ?? connectionStats?.avg ?? authStats?.avg ?? 0;

    return (
        <Card
            title="📊 延迟监控"
            size="small"
            extra={
                <div style={{ display: 'flex', gap: 8 }}>
                    <Badge
                        status={getLatencyStatus(badgeAvg)}
                        text={`${badgeAvg.toFixed(1)}ms`}
                    />
                    {pendingCount > 0 && (
                        <Badge count={pendingCount} title="待确认请求" />
                    )}
                </div>
            }
        >
            <Row gutter={[16, 16]}>
                {stats && (
                    <>
                        <Col span={8}>
                            <Statistic
                                title="ACK 平均"
                                value={stats.avg}
                                precision={2}
                                suffix="ms"
                                valueStyle={{ color: stats.avg < 50 ? '#3f8600' : '#cf1322' }}
                                prefix={<ThunderboltOutlined />}
                            />
                        </Col>
                        <Col span={8}>
                            <Statistic
                                title="ACK P95"
                                value={stats.p95}
                                precision={2}
                                suffix="ms"
                            />
                        </Col>
                        <Col span={8}>
                            <Statistic
                                title="ACK P99"
                                value={stats.p99}
                                precision={2}
                                suffix="ms"
                            />
                        </Col>
                    </>
                )}
                {connectionStats && (
                    <Col span={8}>
                        <Statistic
                            title="连接平均"
                            value={connectionStats.avg}
                            precision={2}
                            suffix="ms"
                        />
                    </Col>
                )}
                {authStats && (
                    <Col span={8}>
                        <Statistic
                            title="认证平均"
                            value={authStats.avg}
                            precision={2}
                            suffix="ms"
                        />
                    </Col>
                )}
                <Col span={8}>
                    <Statistic
                        title="ACK 样本"
                        value={stats?.count ?? 0}
                        prefix={<CheckCircleOutlined />}
                    />
                </Col>
            </Row>
            <Row style={{ marginTop: 16 }} gutter={8}>
                <Col>
                    <Button size="small" onClick={handlePrintReport}>
                        打印报告到控制台
                    </Button>
                </Col>
                <Col>
                    <Button size="small" danger onClick={handleReset}>
                        重置数据
                    </Button>
                </Col>
            </Row>
        </Card>
    );
}
