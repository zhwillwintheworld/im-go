/**
 * 获取 UTC+8 时间字符串（精确到毫秒）
 * 格式：YYYY-MM-DD HH:mm:ss.SSS
 */
export function getUTC8TimeString(): string {
    const now = new Date();
    const utc8Time = new Date(now.getTime() + 8 * 60 * 60 * 1000);
    return utc8Time.toISOString().replace('T', ' ').replace('Z', '');
}

/**
 * 获取当前时间戳（毫秒）
 */
export function now(): number {
    return Date.now();
}
