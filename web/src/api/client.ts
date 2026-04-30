import type { ApiError } from './types';
import { HttpStatus } from './types';

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || '.';

type QueryParams = Record<string, string | number | boolean>;

type RequestOptions = {
    params?: QueryParams;
    headers?: HeadersInit;
    timeoutMs?: number;
};

type RequestConfig = QueryParams | RequestOptions;

/**
 * 获取认证 Store（延迟导入以避免循环依赖）
 */
let getAuthStore: (() => { token: string | null; logout: () => void }) | null = null;

export function setAuthStoreGetter(getter: () => { token: string | null; logout: () => void }) {
    getAuthStore = getter;
}

/**
 * 全局错误处理
 */
const handleError = (error: ApiError) => {
    console.error('API Error:', error);

    // 401 未授权，调用 store 的 logout
    if (error.code === HttpStatus.UNAUTHORIZED) {
        if (getAuthStore) {
            const store = getAuthStore();
            store.logout();
        }
    }
};

/**
 * 处理响应
 */
async function handleResponse<T>(response: Response): Promise<T> {
    const contentType = response.headers.get('content-type');
    const isJson = contentType?.includes('application/json');

    let data: unknown;
    if (isJson) {
        data = await response.json();
    } else {
        data = await response.text();
    }

    if (!response.ok) {
        const error: ApiError = {
            code: response.status,
            message: (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string')
                ? data.message
                : (typeof data === 'string' ? data : response.statusText),
        };

        handleError(error);
        throw error;
    }

    // 如果是标准的 ApiResponse 格式，返回 data 字段
    if (data && typeof data === 'object' && 'data' in data) {
        return data.data as T;
    }

    return data as T;
}

function normalizeOptions(options?: RequestConfig): RequestOptions {
    if (!options) {
        return {};
    }

    if ('params' in options || 'headers' in options || 'timeoutMs' in options) {
        return options as RequestOptions;
    }

    return { params: options as QueryParams };
}

/**
 * 发送请求
 */
async function request<T>(
    method: string,
    path: string,
    body?: BodyInit,
    options?: RequestConfig
): Promise<T> {
    const requestOptions = normalizeOptions(options);

    // 构建 URL
    const searchParams = requestOptions.params ? new URLSearchParams(
        Object.entries(requestOptions.params).map(([k, v]) => [k, String(v)])
    ).toString() : '';
    const url = `${API_BASE_URL}${path}${searchParams ? `?${searchParams}` : ''}`;

    // 构建请求头
    const headers = new Headers();

    // 只在有 body 时设置 Content-Type
    if (body) {
        headers.set('Content-Type', 'application/json');
    }

    // 添加 Authorization - 从 zustand store 获取 token
    if (typeof window !== 'undefined' && getAuthStore) {
        const store = getAuthStore();
        if (store.token) {
            headers.set('Authorization', `Bearer ${store.token}`);
        }
    }

    // 合并调用方传入的请求头，用于危险操作确认等场景
    if (requestOptions.headers) {
        new Headers(requestOptions.headers).forEach((value, key) => {
            headers.set(key, value);
        });
    }

    const controller = requestOptions.timeoutMs && requestOptions.timeoutMs > 0
        ? new AbortController()
        : undefined;
    const timeoutId = controller && requestOptions.timeoutMs
        ? setTimeout(() => controller.abort(), requestOptions.timeoutMs)
        : undefined;

    try {
        // 发送请求
        const response = await fetch(url.toString(), {
            method,
            headers,
            body,
            signal: controller?.signal,
        });

        return await handleResponse<T>(response);
    } catch (error) {
        if (error instanceof Error && error.name === 'AbortError') {
            const apiError: ApiError = {
                code: HttpStatus.REQUEST_TIMEOUT,
                message: `Request timed out after ${requestOptions.timeoutMs}ms`,
            };
            handleError(apiError);
            throw apiError;
        }
        throw error;
    } finally {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
    }
}

/**
 * API 客户端 - 基础 HTTP 方法
 */
export const apiClient = {
    /**
     * GET 请求
     */
    get: <T>(path: string, options?: RequestConfig): Promise<T> =>
        request<T>('GET', path, undefined, options),

    /**
     * POST 请求
     */
    post: <T>(path: string, data?: unknown, options?: RequestConfig): Promise<T> =>
        request<T>('POST', path, data ? JSON.stringify(data) : undefined, options),

    /**
     * PUT 请求
     */
    put: <T>(path: string, data?: unknown, options?: RequestConfig): Promise<T> =>
        request<T>('PUT', path, data ? JSON.stringify(data) : undefined, options),

    /**
     * DELETE 请求
     */
    delete: <T>(path: string, options?: RequestConfig): Promise<T> =>
        request<T>('DELETE', path, undefined, options),

    /**
     * PATCH 请求
     */
    patch: <T>(path: string, data?: unknown, options?: RequestConfig): Promise<T> =>
        request<T>('PATCH', path, data ? JSON.stringify(data) : undefined, options),
};
