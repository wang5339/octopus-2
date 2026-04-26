import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Locale = 'zh-Hans' | 'zh-Hant' | 'en';

function normalizeLocale(locale: unknown): Locale {
    switch (locale) {
        // 兼容旧版本持久化到 localStorage 的下划线格式，避免升级后取不到语言包。
        case 'zh_hans':
        case 'zh-Hans':
            return 'zh-Hans';
        case 'zh_hant':
        case 'zh-Hant':
            return 'zh-Hant';
        case 'en':
            return 'en';
        default:
            return 'zh-Hans';
    }
}

interface SettingState {
    locale: Locale;
    logPageSize: number;
    setLocale: (locale: Locale) => void;
    setLogPageSize: (size: number) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'zh-Hans',
            logPageSize: 50,
            setLocale: (locale) => set({ locale: normalizeLocale(locale) }),
            setLogPageSize: (size) => set({ logPageSize: size }),
        }),
        {
            name: 'octopus-settings',
            merge: (persistedState, currentState) => {
                const persisted = (persistedState ?? {}) as Partial<SettingState> & {
                    locale?: unknown;
                    logPageSize?: unknown;
                };
                return {
                    ...currentState,
                    locale: normalizeLocale(persisted.locale),
                    logPageSize: typeof persisted.logPageSize === 'number' && persisted.logPageSize > 0
                        ? persisted.logPageSize
                        : currentState.logPageSize,
                };
            },
        }
    )
);
