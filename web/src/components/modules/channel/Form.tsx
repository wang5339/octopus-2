import { AutoGroupType, ChannelType, type Channel, useFetchModel, useTestChannelModelsByConfig, type TestModelResult } from '@/api/endpoints/channel';
import { useProviders } from '@/api/endpoints/providers';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';
import { X, Plus, HelpCircle, CheckCircle2, XCircle, Loader2, Check, Search } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';

export interface ChannelKeyFormItem {
    id?: number;
    enabled: boolean;
    channel_key: string;
    status_code?: number;
    last_use_time_stamp?: number;
    total_cost?: number;
    remark?: string;
}

export interface ChannelFormData {
    name: string;
    type: ChannelType;
    base_urls: Channel['base_urls'];
    custom_header: Channel['custom_header'];
    channel_proxy: string;
    param_override: string;
    keys: ChannelKeyFormItem[];
    model: string;
    custom_model: string;
    enabled: boolean;
    proxy: boolean;
    auto_sync: boolean;
    auto_group: AutoGroupType;
    match_regex: string;
    upstream_model_update_ignored_models: string;
}

export interface ChannelFormProps {
    formData: ChannelFormData;
    onFormDataChange: (data: ChannelFormData) => void;
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
    isPending: boolean;
    submitText: string;
    pendingText: string;
    onCancel?: () => void;
    cancelText?: string;
    idPrefix?: string;
    channelId?: number;
}

function getErrorMessage(error: unknown): string | undefined {
    if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
        return error.message;
    }
    return undefined;
}

import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";

export function ChannelForm({
    formData,
    onFormDataChange,
    onSubmit,
    isPending,
    submitText,
    pendingText,
    onCancel,
    cancelText,
    idPrefix = 'channel',
}: ChannelFormProps) {
    const t = useTranslations('channel.form');
    const tModels = useTranslations('channel.models');

    // Fetch providers for auto-fill base_url
    const { data: providers } = useProviders();

    // Test state
    const testByConfig = useTestChannelModelsByConfig();
    const [isTesting, setIsTesting] = useState(false);
    const [testResults, setTestResults] = useState<Map<string, TestModelResult>>(new Map());
    const [dryRun, setDryRun] = useState(true);

    // Ensure the form always shows at least 1 row for base_urls / keys / custom_header.
    // This avoids "empty list" UI and also keeps URL + APIKEY layout consistent.
    useEffect(() => {
        if (!formData.base_urls || formData.base_urls.length === 0) {
            onFormDataChange({ ...formData, base_urls: [{ url: '', delay: 0 }] });
            return;
        }
        if (!formData.keys || formData.keys.length === 0) {
            onFormDataChange({ ...formData, keys: [{ enabled: true, channel_key: '' }] });
            return;
        }
        if (!formData.custom_header || formData.custom_header.length === 0) {
            onFormDataChange({ ...formData, custom_header: [{ header_key: '', header_value: '' }] });
        }
    }, [formData, onFormDataChange]);

    // Auto-fill base_url when type changes and base_url is empty
    useEffect(() => {
        if (!providers) return;

        const provider = providers.find((p) => p.channel_type === formData.type);
        // Only auto-fill if there's exactly one base_url and it's empty
        if (provider && formData.base_urls.length === 1 && formData.base_urls[0].url === '') {
            onFormDataChange({
                ...formData,
                base_urls: [{ url: provider.base_url, delay: 0 }],
            });
        }
    }, [formData, onFormDataChange, providers]);

    const autoModels = formData.model
        ? formData.model.split(',').map((m) => m.trim()).filter(Boolean)
        : [];
    const customModels = formData.custom_model
        ? formData.custom_model.split(',').map((m) => m.trim()).filter(Boolean)
        : [];
    const [fetchedModels, setFetchedModels] = useState<string[]>([]);
    const [inputValue, setInputValue] = useState('');
    const inputRef = useRef<HTMLInputElement>(null);
    const [showModelSelectDialog, setShowModelSelectDialog] = useState(false);
    const [dialogSelectedModels, setDialogSelectedModels] = useState<Set<string>>(new Set());

    const fetchModel = useFetchModel();

    const effectiveKey =
        formData.keys.find((k) => k.enabled && k.channel_key.trim())?.channel_key.trim() || '';

    const updateModels = (nextAuto: string[], nextCustom: string[]) => {
        const model = nextAuto.join(',');
        const custom_model = nextCustom.join(',');
        if (formData.model === model && formData.custom_model === custom_model) return;
        onFormDataChange({ ...formData, model, custom_model });
    };





    const handleConfirmModelSelect = () => {
        const selected = Array.from(dialogSelectedModels);
        // 从弹框选择的模型按“手动模型”处理，保持视觉样式一致
        const newCustom = Array.from(new Set([...customModels, ...selected]));
        updateModels(autoModels, newCustom);
        setShowModelSelectDialog(false);
    };

    const handleAddModel = (model: string) => {
        const trimmedModel = model.trim();
        if (trimmedModel && !customModels.includes(trimmedModel) && !autoModels.includes(trimmedModel)) {
            updateModels(autoModels, [...customModels, trimmedModel]);
        }
        setInputValue('');
    };

    const handleRemoveAutoModel = (model: string) => {
        updateModels(autoModels.filter(m => m !== model), customModels);
    };

    const handleRemoveCustomModel = (model: string) => {
        updateModels(autoModels, customModels.filter(m => m !== model));
    };

    const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (inputValue.trim()) handleAddModel(inputValue);
        }
    };

    const handleAddKey = () => {
        onFormDataChange({
            ...formData,
            keys: [...formData.keys, { enabled: true, channel_key: '' }],
        });
    };

    const handleUpdateKey = (idx: number, patch: Partial<ChannelKeyFormItem>) => {
        const next = formData.keys.map((k, i) => (i === idx ? { ...k, ...patch } : k));
        onFormDataChange({ ...formData, keys: next });
    };

    const handleRemoveKey = (idx: number) => {
        const curr = formData.keys ?? [];
        if (curr.length <= 1) return;
        const next = curr.filter((_, i) => i !== idx);
        onFormDataChange({ ...formData, keys: next });
    };

    const handleAddBaseUrl = () => {
        onFormDataChange({
            ...formData,
            base_urls: [...(formData.base_urls ?? []), { url: '', delay: 0 }],
        });
    };

    const handleUpdateBaseUrl = (idx: number, patch: Partial<Channel['base_urls'][number]>) => {
        const next = (formData.base_urls ?? []).map((u, i) => (i === idx ? { ...u, ...patch } : u));
        onFormDataChange({ ...formData, base_urls: next });
    };

    const handleRemoveBaseUrl = (idx: number) => {
        const curr = formData.base_urls ?? [];
        if (curr.length <= 1) return;
        onFormDataChange({ ...formData, base_urls: curr.filter((_, i) => i !== idx) });
    };

    const handleAddHeader = () => {
        onFormDataChange({
            ...formData,
            custom_header: [...(formData.custom_header ?? []), { header_key: '', header_value: '' }],
        });
    };

    const handleUpdateHeader = (idx: number, patch: Partial<Channel['custom_header'][number]>) => {
        const next = (formData.custom_header ?? []).map((h, i) => (i === idx ? { ...h, ...patch } : h));
        onFormDataChange({ ...formData, custom_header: next });
    };

    const handleRemoveHeader = (idx: number) => {
        const curr = formData.custom_header ?? [];
        if (curr.length <= 1) return;
        onFormDataChange({ ...formData, custom_header: curr.filter((_, i) => i !== idx) });
    };

    // All models (auto + custom)
    const allModels = [
        ...autoModels,
        ...customModels,
    ];

    const handleTestModels = async (models: string[]) => {
        if (models.length === 0 || isTesting) return;
        const hasBaseUrl = formData.base_urls?.some((u) => u.url.trim());
        const hasKey = formData.keys?.some((k) => k.channel_key.trim());
        if (!hasBaseUrl || !hasKey) {
            toast.warning(t('testNeedBaseUrlAndKey'));
            return;
        }
        if (!dryRun && !window.confirm(t('realRequestConfirm', { count: models.length }))) {
            return;
        }
        setIsTesting(true);
        try {
            const results = await testByConfig.mutateAsync({
                type: formData.type,
                base_urls: formData.base_urls.filter((u) => u.url.trim()),
                keys: formData.keys.filter((k) => k.channel_key.trim()).map((k) => ({ enabled: k.enabled, channel_key: k.channel_key.trim() })),
                proxy: formData.proxy,
                channel_proxy: formData.channel_proxy?.trim() || null,
                param_override: formData.param_override?.trim() || null,
                custom_header: formData.custom_header?.filter((h) => h.header_key.trim()) || [],
                models,
                dry_run: dryRun,
            });
            const map = new Map<string, TestModelResult>();
            for (const r of results) map.set(r.model, r);
            setTestResults(map);
        } catch (error) {
            toast.error(t('testFailed'), { description: getErrorMessage(error) });
        } finally {
            setIsTesting(false);
        }
    };

    const handleTestFirst = () => {
        if (allModels.length > 0) handleTestModels([allModels[0]]);
    };

    const handleTestAll = () => {
        handleTestModels(allModels);
    };

    // Provider preset quick-select
    const handleProviderPreset = (providerName: string) => {
        if (!providers) return;
        const provider = providers.find((p) => p.name === providerName);
        if (!provider) return;
        onFormDataChange({
            ...formData,
            type: provider.channel_type as ChannelType,
            base_urls: [{ url: provider.base_url, delay: 0 }],
        });
    };

    const namePlaceholder = (() => {
        if (!providers) return t('namePlaceholder');
        const currentUrl = formData.base_urls?.[0]?.url?.trim();
        const p = providers.find((p) => currentUrl && p.base_url === currentUrl);
        return p ? `${t('namePlaceholderPrefix')}${p.name}` : t('namePlaceholder');
    })();

    return (
        <>
        <form onSubmit={onSubmit} className="space-y-4 px-1">
            {/* Provider 快速预设选择 */}
            {providers && providers.length > 0 && (
                <div className="space-y-2">
                    <label className="text-sm font-medium text-card-foreground">{t('providerPreset')}</label>
                    <Select onValueChange={handleProviderPreset}>
                        <SelectTrigger className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                            <SelectValue placeholder={t('providerPresetPlaceholder')} />
                        </SelectTrigger>
                        <SelectContent className="rounded-xl">
                            {providers.map((p) => (
                                <SelectItem key={`${p.name}-${p.channel_type}`} className="rounded-xl" value={p.name}>
                                    {p.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            )}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                    <label htmlFor={`${idPrefix}-name`} className="text-sm font-medium text-card-foreground">
                        {t('name')}
                    </label>
                    <Input
                        className='rounded-xl'
                        id={`${idPrefix}-name`}
                        type="text"
                        value={formData.name}
                        onChange={(event) => onFormDataChange({ ...formData, name: event.target.value })}
                        placeholder={namePlaceholder}
                        required
                    />
                </div>

                <div className="space-y-2">
                    <label htmlFor={`${idPrefix}-type`} className="text-sm font-medium text-card-foreground">
                        {t('type')}
                    </label>
                    <Select
                        value={String(formData.type)}
                        onValueChange={(value) => onFormDataChange({ ...formData, type: Number(value) as ChannelType })}
                    >
                        <SelectTrigger id={`${idPrefix}-type`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent className='rounded-xl'>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIChat)}>{t('typeOpenAIChat')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIResponse)}>{t('typeOpenAIResponse')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Anthropic)}>{t('typeAnthropic')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Gemini)}>{t('typeGemini')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Volcengine)}>{t('typeVolcengine')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIEmbedding)}>{t('typeOpenAIEmbedding')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIImageGeneration)}>{t('typeOpenAIImageGeneration')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Zen)}>{t('typeZen')}</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-1">
                        <label className="text-sm font-medium text-card-foreground">
                            {t('baseUrls')} {formData.base_urls.length > 0 ? `(${formData.base_urls.length})` : ''}
                        </label>
                        <TooltipProvider>
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <HelpCircle className="size-3.5 text-muted-foreground cursor-help" />
                                </TooltipTrigger>
                                <TooltipContent>
                                    {t('baseUrlTooltip')}
                                </TooltipContent>
                            </Tooltip>
                        </TooltipProvider>
                    </div>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleAddBaseUrl}
                        className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                    >
                        <Plus className="h-3 w-3 mr-1" />
                        {t('add')}
                    </Button>
                </div>
                <div className="space-y-2">
                    {(formData.base_urls ?? []).map((u, idx) => (
                        <div key={`baseurl-${idx}`} className="flex items-center gap-2">
                            <Input
                                id={`${idPrefix}-base-${idx}`}
                                type="url"
                                value={u.url}
                                onChange={(e) => handleUpdateBaseUrl(idx, { url: e.target.value })}
                                placeholder={t('baseUrlUrl')}
                                required={idx === 0}
                                className="rounded-xl flex-1"
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveBaseUrl(idx)}
                                disabled={(formData.base_urls ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive disabled:opacity-40 hover:bg-transparent"
                                title="Remove"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">
                        {t('apiKey')} {formData.keys.length > 0 ? `(${formData.keys.length})` : ''}
                    </label>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleAddKey}
                        className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                    >
                        <Plus className="h-3 w-3 mr-1" />
                        {t('add')}
                    </Button>
                </div>

                <div className="space-y-2">
                    {(formData.keys ?? []).map((k, idx) => (
                        <div key={k.id ?? `new-${idx}`} className="flex items-center gap-2">
                            <Input
                                type="text"
                                value={k.channel_key}
                                onChange={(e) => handleUpdateKey(idx, { channel_key: e.target.value })}
                                placeholder={t('apiKey')}
                                required={idx === 0}
                                className="rounded-xl flex-1"
                            />
                            <Input
                                type="text"
                                value={k.remark ?? ''}
                                onChange={(e) => handleUpdateKey(idx, { remark: e.target.value })}
                                placeholder={t('remark')}
                                className="rounded-xl w-32"
                            />
                            <Switch
                                checked={k.enabled}
                                onCheckedChange={(checked) => handleUpdateKey(idx, { enabled: checked })}
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveKey(idx)}
                                disabled={(formData.keys ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                title="Remove"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">{t('model')}</label>
                    <div className="flex items-center gap-1">
                        {(effectiveKey && formData.base_urls?.[0]?.url) && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                disabled={fetchModel.isPending}
                                onClick={() => {
                                    // 每次打开都按当前已选模型重建预选状态，避免残留历史选择
                                    setDialogSelectedModels(new Set(fetchedModels.filter((model) => allModels.includes(model))));
                                    // 立即打开弹框
                                    setShowModelSelectDialog(true);
                                    // 如果还没有数据，则在后台请求
                                    if (fetchedModels.length === 0) {
                                        fetchModel.mutate(
                                            {
                                                type: formData.type,
                                                base_urls: formData.base_urls,
                                                keys: formData.keys
                                                    .filter((k) => k.channel_key.trim())
                                                    .map((k) => ({ enabled: k.enabled, channel_key: k.channel_key.trim() })),
                                                proxy: formData.proxy,
                                                channel_proxy: formData.channel_proxy?.trim() || null,
                                                match_regex: formData.match_regex.trim() || null,
                                                custom_header: formData.custom_header?.filter((h) => h.header_key.trim()) || [],
                                            },
                                            {
                                                onSuccess: (data) => {
                                                    if (data && data.length > 0) {
                                                        const nextFetched = Array.from(new Set(data.map((m) => m.trim()).filter(Boolean)));
                                                        setFetchedModels(nextFetched);
                                                        // 拉取后再基于当前模型列表同步一次预选
                                                        setDialogSelectedModels(new Set(nextFetched.filter((model) => allModels.includes(model))));
                                                    } else {
                                                        toast.warning(t('modelRefreshEmpty'));
                                                    }
                                                },
                                                onError: (error) => {
                                                    const errorMessage = error instanceof Error ? error.message : String(error);
                                                    toast.error(t('modelRefreshFailed'), { description: errorMessage });
                                                },
                                            }
                                        );
                                    }
                                }}
                                className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent gap-1"
                            >
                                <Search className="h-3 w-3" />
                                {t('selectModels')}
                            </Button>
                        )}
                    </div>
                </div>
                <input type="hidden" value={formData.model} />

                <div className="relative">
                    <Input
                        ref={inputRef}
                        id={`${idPrefix}-model-custom`}
                        type="text"
                        value={inputValue}
                        onChange={(e) => setInputValue(e.target.value)}
                        onKeyDown={handleInputKeyDown}
                        placeholder={t('modelCustomPlaceholder')}
                        className="pr-10 rounded-xl"
                    />
                    {inputValue.trim() && !customModels.includes(inputValue.trim()) && !autoModels.includes(inputValue.trim()) && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => handleAddModel(inputValue)}
                            className="absolute rounded-lg right-1 top-1/2 -translate-y-1/2 h-7 w-7 p-0 text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
                            title={t('modelAdd')}
                        >
                            <Plus className="size-4" />
                        </Button>
                    )}
                </div>

                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <label className="text-xs font-medium text-card-foreground">
                            {t('modelSelected')} {(autoModels.length + customModels.length) > 0 && `(${autoModels.length + customModels.length})`}
                        </label>
                        {(autoModels.length + customModels.length) > 0 && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    updateModels([], []);
                                }}
                                className="h-6 px-2 text-xs text-muted-foreground/50 hover:text-muted-foreground hover:bg-transparent"
                            >
                                {t('modelClearAll')}
                            </Button>
                        )}
                    </div>
                    <div className="rounded-xl border border-border bg-muted/30 p-2.5 max-h-40 min-h-12 overflow-y-auto">
                        {(autoModels.length + customModels.length) > 0 ? (
                            <div className="flex flex-wrap gap-1.5">
                                {autoModels.map((model) => (
                                    <Badge key={model} className="bg-primary hover:bg-primary/90">
                                        {model}
                                        <button
                                            type="button"
                                            onClick={() => handleRemoveAutoModel(model)}
                                            className="ml-1 rounded-sm opacity-70 hover:opacity-100 focus:outline-none focus:ring-1 focus:ring-ring"
                                        >
                                            <X className="h-3 w-3" />
                                        </button>
                                    </Badge>
                                ))}
                                {customModels.map((model) => (
                                    <Badge key={model} className="bg-primary hover:bg-primary/90">
                                        {model}
                                        <button
                                            type="button"
                                            onClick={() => handleRemoveCustomModel(model)}
                                            className="ml-1 rounded-sm opacity-70 hover:opacity-100 focus:outline-none focus:ring-1 focus:ring-ring"
                                        >
                                            <X className="h-3 w-3" />
                                        </button>
                                    </Badge>
                                ))}
                            </div>
                        ) : (
                            <div className="flex items-center justify-center h-8 text-xs text-muted-foreground">
                                {t('modelNoSelected')}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <Accordion type="single" collapsible className="w-full border rounded-xl bg-card">
                <AccordionItem value="advanced" className="border-none">
                    <AccordionTrigger className="text-sm font-medium text-card-foreground py-3 px-4 hover:no-underline hover:bg-muted/30 rounded-xl transition-colors">
                        {t('advanced')}
                    </AccordionTrigger>
                    <AccordionContent className="pt-4 px-4 pb-4 space-y-4 border-t">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <label htmlFor={`${idPrefix}-auto-group`} className="text-sm font-medium text-card-foreground">
                                    {t('autoGroup')}
                                </label>
                                <Select
                                    value={String(formData.auto_group)}
                                    onValueChange={(value) => onFormDataChange({ ...formData, auto_group: Number(value) as AutoGroupType })}
                                >
                                    <SelectTrigger id={`${idPrefix}-auto-group`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent className='rounded-xl'>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.None)}>{t('autoGroupNone')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Fuzzy)}>{t('autoGroupFuzzy')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Exact)}>{t('autoGroupExact')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Regex)}>{t('autoGroupRegex')}</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="space-y-2">
                                <label htmlFor={`${idPrefix}-channel-proxy`} className="text-sm font-medium text-card-foreground">
                                    {t('channelProxy')}
                                </label>
                                <Input
                                    id={`${idPrefix}-channel-proxy`}
                                    type="text"
                                    value={formData.channel_proxy}
                                    onChange={(e) => onFormDataChange({ ...formData, channel_proxy: e.target.value })}
                                    placeholder={t('channelProxyPlaceholder')}
                                    className="rounded-xl"
                                />
                            </div>
                        </div>

                        <div className="space-y-2">
                            <div className="flex items-center justify-between">
                                <label className="text-sm font-medium text-card-foreground">
                                    {t('customHeader')} {formData.custom_header.length > 0 ? `(${formData.custom_header.length})` : ''}
                                </label>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={handleAddHeader}
                                    className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                                >
                                    <Plus className="h-3 w-3 mr-1" />
                                    {t('customHeaderAdd')}
                                </Button>
                            </div>
                            <div className="space-y-2">
                                {(formData.custom_header ?? []).map((h, idx) => (
                                    <div key={`hdr-${idx}`} className="flex items-center gap-2">
                                        <Input
                                            type="text"
                                            value={h.header_key}
                                            onChange={(e) => handleUpdateHeader(idx, { header_key: e.target.value })}
                                            placeholder={t('customHeaderKey')}
                                            className="rounded-xl flex-1"
                                        />
                                        <Input
                                            type="text"
                                            value={h.header_value}
                                            onChange={(e) => handleUpdateHeader(idx, { header_value: e.target.value })}
                                            placeholder={t('customHeaderValue')}
                                            className="rounded-xl flex-1"
                                        />
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => handleRemoveHeader(idx)}
                                            disabled={(formData.custom_header ?? []).length <= 1}
                                            className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                            title="Remove"
                                        >
                                            <X className="h-4 w-4" />
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        </div>

                        <div className="space-y-2">
                            <label htmlFor={`${idPrefix}-match-regex`} className="text-sm font-medium text-card-foreground">
                                {t('matchRegex')}
                            </label>
                            <Input
                                id={`${idPrefix}-match-regex`}
                                type="text"
                                value={formData.match_regex}
                                onChange={(e) => onFormDataChange({ ...formData, match_regex: e.target.value })}
                                placeholder={t('matchRegexPlaceholder')}
                                className="rounded-xl"
                            />
                        </div>

                        <div className="space-y-2">
                            <label htmlFor={`${idPrefix}-upstream-ignored`} className="text-sm font-medium text-card-foreground">
                                {t('upstreamIgnoredModels')}
                            </label>
                            <textarea
                                id={`${idPrefix}-upstream-ignored`}
                                value={formData.upstream_model_update_ignored_models}
                                onChange={(e) => onFormDataChange({ ...formData, upstream_model_update_ignored_models: e.target.value })}
                                placeholder={t('upstreamIgnoredModelsPlaceholder')}
                                className="min-h-20 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            />
                            <p className="text-xs text-muted-foreground">
                                {t('upstreamIgnoredModelsHelp')}
                            </p>
                        </div>

                        <div className="space-y-2">
                            <label htmlFor={`${idPrefix}-param-override`} className="text-sm font-medium text-card-foreground">
                                {t('paramOverride')}
                            </label>
                            <textarea
                                id={`${idPrefix}-param-override`}
                                value={formData.param_override}
                                onChange={(e) => onFormDataChange({ ...formData, param_override: e.target.value })}
                                placeholder={t('paramOverridePlaceholder')}
                                className="min-h-28 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            />
                        </div>
                    </AccordionContent>
                </AccordionItem>
            </Accordion>

            <div className="flex flex-wrap items-center justify-between gap-4 p-4 rounded-xl bg-muted/20 border border-border/50">
                <label className="flex items-center gap-2 cursor-pointer">
                    <Switch
                        checked={formData.enabled}
                        onCheckedChange={(checked) => onFormDataChange({ ...formData, enabled: checked })}
                    />
                    <span className="text-sm font-medium text-card-foreground">{t('enabled')}</span>
                </label>
                <div className="flex items-center gap-6">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <Switch
                            checked={formData.proxy}
                            onCheckedChange={(checked) => onFormDataChange({ ...formData, proxy: checked })}
                        />
                        <span className="text-sm text-card-foreground">{t('proxy')}</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                        <Switch
                            checked={formData.auto_sync}
                            onCheckedChange={(checked) => onFormDataChange({ ...formData, auto_sync: checked })}
                        />
                        <span className="text-sm text-card-foreground">{t('autoSync')}</span>
                    </label>
                </div>
            </div>

            <div className={`flex flex-col gap-3 pt-2 ${onCancel ? 'sm:flex-row' : ''}`}>
                <label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer">
                    <Switch
                        checked={dryRun}
                        onCheckedChange={setDryRun}
                    />
                    <span>{t('dryRun')}</span>
                </label>
                {onCancel && cancelText && (
                    <Button
                        type="button"
                        variant="secondary"
                        onClick={onCancel}
                        className="w-full sm:flex-1 rounded-2xl h-12"
                    >
                        {cancelText}
                    </Button>
                )}
                <div className="flex gap-2 w-full sm:flex-1">
                    <Button
                        type="button"
                        variant="outline"
                        disabled={isTesting || allModels.length === 0}
                        onClick={handleTestFirst}
                        className="flex-1 rounded-2xl h-12"
                        title={t('testFirstTitle')}
                    >
                        {isTesting ? (
                            <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                        ) : null}
                        {t('testFirst')}
                    </Button>
                    <Button
                        type="button"
                        variant="outline"
                        disabled={isTesting || allModels.length === 0}
                        onClick={handleTestAll}
                        className="flex-1 rounded-2xl h-12"
                        title={t('testAllTitle')}
                    >
                        {isTesting ? (
                            <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                        ) : null}
                        {t('testAll')}
                    </Button>
                    <Button
                        type="submit"
                        disabled={isPending}
                        className="flex-3 rounded-2xl h-12"
                    >
                        {isPending ? pendingText : submitText}
                    </Button>
                </div>
            </div>

            {/* 测试结果摘要 */}
            {testResults.size > 0 && (
                <div className="rounded-xl border border-border bg-muted/20 p-3 space-y-2">
                    <div className="text-xs font-medium text-card-foreground">{t('testResultTitle')}</div>
                    <div className="space-y-1 max-h-40 overflow-y-auto">
                        {Array.from(testResults.entries()).map(([model, result]) => (
                            <div key={model} className="flex items-center gap-2 text-xs">
                                {result.passed ? (
                                    <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />
                                ) : (
                                    <XCircle className="h-3.5 w-3.5 text-red-500 shrink-0" />
                                )}
                                <span className="font-mono flex-1 truncate">{model}</span>
                                {result.delay !== undefined && (
                                    <span className="text-muted-foreground">{result.delay}ms</span>
                                )}
                                {result.error && (
                                    <span className="text-red-500 truncate max-w-32" title={result.error}>{result.error}</span>
                                )}
                            </div>
                        ))}
                    </div>
                    <div className="text-xs text-muted-foreground">
                        {t('testResultSummary', {
                            total: testResults.size,
                            passed: Array.from(testResults.values()).filter((r) => r.passed).length,
                        })}
                    </div>
                </div>
            )}
        </form>
        {/* 选择模型弹框 - 放在 form 外避免事件冒泡关闭外层弹框 */}
        <Dialog open={showModelSelectDialog} onOpenChange={setShowModelSelectDialog}>
            <DialogContent className="max-w-md flex flex-col max-h-[80vh] overflow-hidden">
                <DialogHeader className="shrink-0">
                    <div className="flex items-center justify-between">
                        <DialogTitle>{t('fetchedModelList')}</DialogTitle>
                        {fetchModel.isPending && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
                    </div>
                </DialogHeader>
                {/* 全选行 */}
                {fetchModel.isPending ? (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                ) : fetchedModels.length === 0 ? (
                    <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
                        {tModels('noModels')}
                    </div>
                ) : (
                    <>
                    <div
                        className="flex items-center gap-2 px-1 py-1 cursor-pointer hover:bg-accent/5 rounded-lg shrink-0"
                        onClick={() => {
                            if (dialogSelectedModels.size === fetchedModels.length) {
                                setDialogSelectedModels(new Set());
                            } else {
                                setDialogSelectedModels(new Set(fetchedModels));
                            }
                        }}
                    >
                        <div className="size-4 shrink-0 rounded border border-primary flex items-center justify-center">
                            {dialogSelectedModels.size === fetchedModels.length && fetchedModels.length > 0 && (
                                <Check className="h-3 w-3 text-primary" />
                            )}
                        </div>
                        <span className="text-sm font-medium">{t('modelSelectAllFetched')}</span>
                    </div>
                <div className="border-t shrink-0" />
                {/* 模型列表 - 直接作为 flex 子项，自身滑动 */}
                <div
                    className="flex-1 min-h-0 overflow-y-auto space-y-0.5 py-1 dialog-model-scrollbar"
                    style={{ scrollbarWidth: 'thin', msOverflowStyle: 'auto' }}
                >
                    {fetchedModels.map((model) => (
                        <div
                            key={model}
                            className="flex items-center gap-2 px-1 py-1.5 cursor-pointer hover:bg-accent/5 rounded-lg"
                            onClick={() => {
                                const next = new Set(dialogSelectedModels);
                                if (next.has(model)) next.delete(model);
                                else next.add(model);
                                setDialogSelectedModels(next);
                            }}
                        >
                            <div className="size-4 shrink-0 rounded border border-primary flex items-center justify-center">
                                {dialogSelectedModels.has(model) && (
                                    <Check className="h-3 w-3 text-primary" />
                                )}
                            </div>
                            <span className="font-mono text-sm">{model}</span>
                        </div>
                    ))}
                    </div>
                    </>
                )}
                <DialogFooter className="shrink-0">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => setShowModelSelectDialog(false)}
                        className="rounded-xl"
                    >
                        {t('selectModelsCancel')}
                    </Button>
                    <Button
                        type="button"
                        onClick={handleConfirmModelSelect}
                        className="rounded-xl"
                    >
                        {t('selectModelsConfirm')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
        </>
    );
}
