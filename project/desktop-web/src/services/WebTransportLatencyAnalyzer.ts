import { getUTC8TimeString } from '@/utils/time';

/**
 * WebTransport 延迟分析器
 * 用于监控和分析消息往返延迟
 */
export class WebTransportLatencyAnalyzer {
    private latencies: number[] = [];
    private timestamps: Map<string, { startTime: number; timeString: string }> = new Map();
    private maxHistorySize: number = 1000;

    /**
     * 记录消息发送时间
     */
    recordSend(reqId: string): void {
        this.timestamps.set(reqId, {
            startTime: performance.now(),
            timeString: getUTC8TimeString()
        });
    }

    /**
     * 记录消息接收时间并计算延迟
     */
    recordReceive(reqId: string): { latency: number; sendTimeString: string; receiveTimeString: string } | null {
        const sendRecord = this.timestamps.get(reqId);
        if (!sendRecord) {
            return null;
        }

        const latency = performance.now() - sendRecord.startTime;
        const receiveTimeString = getUTC8TimeString();
        this.latencies.push(latency);
        this.timestamps.delete(reqId);

        // 限制历史记录大小
        if (this.latencies.length > this.maxHistorySize) {
            this.latencies.shift();
        }

        return {
            latency,
            sendTimeString: sendRecord.timeString,
            receiveTimeString
        };
    }

    /**
     * 获取统计数据
     */
    getStats(): LatencyStats | null {
        if (this.latencies.length === 0) {
            return null;
        }

        const sorted = [...this.latencies].sort((a, b) => a - b);
        const sum = this.latencies.reduce((a, b) => a + b, 0);

        return {
            count: this.latencies.length,
            avg: sum / this.latencies.length,
            min: sorted[0],
            max: sorted[sorted.length - 1],
            p50: sorted[Math.floor(sorted.length * 0.5)],
            p95: sorted[Math.floor(sorted.length * 0.95)],
            p99: sorted[Math.floor(sorted.length * 0.99)],
        };
    }

    /**
     * 获取最近 N 条延迟记录
     */
    getRecentLatencies(count: number = 10): number[] {
        return this.latencies.slice(-count);
    }

    /**
     * 检测异常延迟（超过平均值 2 倍）
     */
    detectAnomalies(): AnomalyResult | null {
        const stats = this.getStats();
        if (!stats) {
            return null;
        }

        const threshold = stats.avg * 2;
        const anomalies = this.latencies.filter(l => l > threshold);

        return {
            threshold,
            count: anomalies.length,
            percentage: (anomalies.length / this.latencies.length) * 100,
            samples: anomalies.slice(-5), // 最近 5 个异常
        };
    }

    /**
     * 清空所有数据
     */
    reset(): void {
        this.latencies = [];
        this.timestamps.clear();
    }

    /**
     * 获取待确认的请求数量（已发送但未收到 ACK）
     */
    getPendingCount(): number {
        return this.timestamps.size;
    }

    /**
     * 打印统计报告
     */
    printReport(): void {
        const stats = this.getStats();
        if (!stats) {
            console.log('[延迟分析] 暂无数据');
            return;
        }

        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log('📊 WebTransport 延迟分析报告');
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log(`📦 样本数量: ${stats.count}`);
        console.log(`📈 平均延迟: ${stats.avg.toFixed(2)}ms`);
        console.log(`⬇️  最小延迟: ${stats.min.toFixed(2)}ms`);
        console.log(`⬆️  最大延迟: ${stats.max.toFixed(2)}ms`);
        console.log(`📊 P50 (中位数): ${stats.p50.toFixed(2)}ms`);
        console.log(`📊 P95: ${stats.p95.toFixed(2)}ms`);
        console.log(`📊 P99: ${stats.p99.toFixed(2)}ms`);

        const anomalies = this.detectAnomalies();
        if (anomalies && anomalies.count > 0) {
            console.log(`⚠️  异常延迟: ${anomalies.count} 次 (${anomalies.percentage.toFixed(1)}%)`);
            console.log(`   阈值: ${anomalies.threshold.toFixed(2)}ms`);
        }

        const pending = this.getPendingCount();
        if (pending > 0) {
            console.log(`⏳ 待确认请求: ${pending}`);
        }
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    }
}

export interface LatencyStats {
    count: number;   // 样本数量
    avg: number;     // 平均延迟
    min: number;     // 最小延迟
    max: number;     // 最大延迟
    p50: number;     // 中位数
    p95: number;     // 95 分位数
    p99: number;     // 99 分位数
}

export interface AnomalyResult {
    threshold: number;     // 异常阈值
    count: number;         // 异常数量
    percentage: number;    // 异常百分比
    samples: number[];     // 异常样本
}

// 导出单例
export const latencyAnalyzer = new WebTransportLatencyAnalyzer();
