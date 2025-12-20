/**
 * 应用配置
 * 从环境变量读取配置，支持 dev / test / prod 三种环境
 */

export const config = {
    /** 当前环境 */
    env: import.meta.env.VITE_ENV || 'development',

    /** Web API 基础地址 */
    apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8082',

    /** Access 服务地址 (WebTransport) */
    accessUrl: import.meta.env.VITE_ACCESS_URL || 'https://localhost:8081',

    /** 日志级别 */
    logLevel: import.meta.env.VITE_LOG_LEVEL || 'debug',

    /** 是否为开发环境 */
    isDev: import.meta.env.VITE_ENV === 'development',

    /** 是否为生产环境 */
    isProd: import.meta.env.VITE_ENV === 'production',

    /** WebTransport 连接地址 */
    get webTransportUrl(): string {
        return `${this.accessUrl}/webtransport`;
    },
} as const;

export default config;
