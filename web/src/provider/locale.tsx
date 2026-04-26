'use client';

import { type ReactNode } from 'react';
import { NextIntlClientProvider } from 'next-intl';
import { useSettingStore, type Locale } from '@/stores/setting';

import zh_HansMessages from '../../public/locale/zh-Hans.json';
import zh_HantMessages from '../../public/locale/zh-Hant.json';
import enMessages from '../../public/locale/en.json';

const messages: Record<Locale, typeof zh_HansMessages> = {
    'zh-Hans': zh_HansMessages,
    'zh-Hant': zh_HantMessages,
    en: enMessages,
};

export function LocaleProvider({ children }: { children: ReactNode }) {
    const { locale } = useSettingStore();

    return (
        <NextIntlClientProvider
            locale={locale}
            messages={messages[locale]}
            timeZone="Asia/Shanghai"
        >
            {children}
        </NextIntlClientProvider>
    );
}

