import { config } from '@/config';

export interface ApiResponse<T = unknown> {
    code: number;
    message: string;
    data: T;
}

// 获取 token
const getToken = (): string | null => {
    return localStorage.getItem('token');
};

// 封装请求方法
async function request<T = unknown>(
    method: string,
    path: string,
    data?: unknown
): Promise<{ data: ApiResponse<T> }> {
    const token = getToken();
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };

    if (token) {
        headers['Authorization'] = token;
    }

    const options: RequestInit = {
        method,
        headers,
    };

    if (data && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
        options.body = JSON.stringify(data);
    }

    const response = await fetch(`${config.apiBaseUrl}/api/v1${path}`, options);
    const json = await response.json();
    return { data: json };
}

// API 客户端
export const apiClient = {
    get: <T = unknown>(path: string) => request<T>('GET', path),
    post: <T = unknown>(path: string, data?: unknown) => request<T>('POST', path, data),
    put: <T = unknown>(path: string, data?: unknown) => request<T>('PUT', path, data),
    delete: <T = unknown>(path: string) => request<T>('DELETE', path),
};
