import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';

/**
 * 获取后端当前版本 Hook
 *
 * 后端: GET /api/v1/version -> string
 */
export function useNowVersion() {
    return useQuery({
        queryKey: ['version', 'now'],
        queryFn: async () => {
            return apiClient.get<string>('/api/v1/version');
        },
        refetchInterval: 3600000, // 1 小时
        refetchOnMount: 'always',
    });
}
