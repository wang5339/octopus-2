import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useTranslations } from 'next-intl';
import { CheckCircle2, XCircle, Loader2, Check, RefreshCw, Wand2 } from 'lucide-react';
import { ChannelType, type Channel, type ModelProtocolOverride, type ModelProtocolDetectResult } from '@/api/endpoints/channel';
import { useApplyChannelModelProtocols, useApplyChannelUpstreamUpdates, useDetectChannelModelProtocols, useDetectChannelUpstreamUpdates, useTestChannelModels, useUpdateChannel } from '@/api/endpoints/channel';
import { toast } from '@/components/common/Toast';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';

function Checkbox({
    checked,
    onChange,
    id,
    ariaLabel,
}: {
    checked: boolean;
    onChange: (checked: boolean) => void;
    id?: string;
    ariaLabel?: string;
}) {
    return (
        <button
            type="button"
            id={id}
            role="checkbox"
            aria-checked={checked}
            aria-label={ariaLabel}
            className="size-4 shrink-0 rounded border border-primary cursor-pointer flex items-center justify-center bg-transparent p-0 transition-colors hover:bg-primary/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            onClick={() => onChange(!checked)}
        >
            {checked && <Check className="h-3 w-3 text-primary" />}
        </button>
    );
}

function getErrorMessage(error: unknown): string | undefined {
    if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
        return error.message;
    }
    return undefined;
}

interface ModelTabContentProps {
    channel: Channel;
}

export function ModelTabContent({ channel }: ModelTabContentProps) {
    const t = useTranslations('channel.models');
    const testModels = useTestChannelModels();
    const detectUpstreamUpdates = useDetectChannelUpstreamUpdates();
    const applyUpstreamUpdates = useApplyChannelUpstreamUpdates();
    const updateChannel = useUpdateChannel();
    const detectModelProtocols = useDetectChannelModelProtocols();
    const applyModelProtocols = useApplyChannelModelProtocols();
    const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
    const [testResults, setTestResults] = useState<Map<string, { passed: boolean; error?: string; delay?: number }>>(new Map());
    const [isTesting, setIsTesting] = useState(false);
    const [protocolResults, setProtocolResults] = useState<Map<string, ModelProtocolDetectResult>>(new Map());
    const [dryRun, setDryRun] = useState(true);

    // 获取所有模型列表（auto + custom）
    const allModels = [
        ...channel.model?.split(',').filter(Boolean) ?? [],
        ...channel.custom_model?.split(',').filter(Boolean) ?? []
    ];
    const pendingAddModels = channel.upstream_model_update_last_detected_models ?? [];
    const pendingRemoveModels = channel.upstream_model_update_last_removed_models ?? [];
    const modelProtocolOverrides = channel.model_protocol_overrides ?? [];
    const protocolOptions = [
        { value: String(ChannelType.OpenAIChat), label: t('modelProtocolOpenAIChat') },
        { value: String(ChannelType.OpenAIResponse), label: t('modelProtocolOpenAIResponse') },
        { value: String(ChannelType.Anthropic), label: t('modelProtocolAnthropic') },
        { value: String(ChannelType.Gemini), label: t('modelProtocolGemini') },
        { value: String(ChannelType.Volcengine), label: t('modelProtocolVolcengine') },
    ];

    const getModelProtocolValue = (modelName: string) => {
        const override = modelProtocolOverrides.find((item) => item.model === modelName);
        return override ? String(override.type) : 'default';
    };

    const handleModelProtocolChange = async (modelName: string, value: string) => {
        const nextOverrides: ModelProtocolOverride[] = modelProtocolOverrides
            .filter((item) => item.model !== modelName && allModels.includes(item.model));

        if (value !== 'default') {
            nextOverrides.push({
                model: modelName,
                type: Number(value) as ChannelType,
            });
        }

        await updateChannel.mutateAsync({
            id: channel.id,
            model_protocol_overrides: nextOverrides,
        });
        toast.success(t('modelProtocolSaved'));
    };

    const getProtocolLabel = (type?: ChannelType | null) => {
        if (type === null || type === undefined) return t('modelProtocolNoRecommendation');
        const option = protocolOptions.find((item) => Number(item.value) === type);
        return option?.label ?? t('modelProtocolNoRecommendation');
    };

    const handleDetectModelProtocols = async () => {
        const modelsToDetect = selectedModels.size > 0 ? Array.from(selectedModels) : allModels;
        if (modelsToDetect.length === 0) return;

        const requestCount = modelsToDetect.length * protocolOptions.length;
        if (!dryRun && !window.confirm(t('realRequestConfirm', { count: requestCount }))) {
            return;
        }

        try {
            const results = await detectModelProtocols.mutateAsync({
                id: channel.id,
                models: modelsToDetect,
                dry_run: dryRun,
            });
            setProtocolResults(new Map(results.map((item) => [item.model, item])));
            toast.success(t('modelProtocolDetectDone', { count: results.length }));
        } catch (error) {
            toast.error(t('modelProtocolDetectFailed'), { description: getErrorMessage(error) });
        }
    };

    const handleApplyRecommendedProtocols = async () => {
        const overrides = Array.from(protocolResults.values())
            .filter((item): item is ModelProtocolDetectResult & { recommended: ChannelType } => item.recommended !== null && item.recommended !== undefined)
            .map((item) => ({ model: item.model, type: item.recommended }));
        if (overrides.length === 0) {
            toast.success(t('modelProtocolNoRecommendation'));
            return;
        }

        await applyModelProtocols.mutateAsync({
            id: channel.id,
            overrides,
        });
        toast.success(t('modelProtocolApplyDone', { count: overrides.length }));
    };

    const handleDetectUpstream = async () => {
        const result = await detectUpstreamUpdates.mutateAsync({ id: channel.id });
        toast.success(t('upstreamDetectDone', {
            add: result.add_models.length,
            remove: result.remove_models.length,
        }));
    };

    const handleApplyUpstream = async (mode: 'add' | 'remove') => {
        await applyUpstreamUpdates.mutateAsync({
            id: channel.id,
            add_models: mode === 'add' ? pendingAddModels : [],
            remove_models: mode === 'remove' ? pendingRemoveModels : [],
        });
        toast.success(mode === 'add' ? t('upstreamApplyAddDone') : t('upstreamApplyRemoveDone'));
    };

    const handleToggleAll = () => {
        if (selectedModels.size === allModels.length) {
            setSelectedModels(new Set());
        } else {
            setSelectedModels(new Set(allModels));
        }
    };

    const handleTest = async (models?: string[]) => {
        const modelsToTest = models ?? Array.from(selectedModels);
        if (modelsToTest.length === 0) return;

        if (!dryRun && !window.confirm(t('realRequestConfirm', { count: modelsToTest.length }))) {
            return;
        }

        setIsTesting(true);
        try {
            const results = await testModels.mutateAsync({
                channel_id: channel.id,
                models: modelsToTest,
                dry_run: dryRun,
            });

            // Convert array results to Map
            const resultsMap = new Map<string, { passed: boolean; error?: string; delay?: number }>();
            for (const result of results) {
                resultsMap.set(result.model, {
                    passed: result.passed,
                    error: result.error,
                    delay: result.delay,
                });
            }
            setTestResults(resultsMap);
        } catch (error) {
            toast.error(t('testFailed'), { description: getErrorMessage(error) });
        } finally {
            setIsTesting(false);
        }
    };

    const handleTestFirst = () => {
        if (allModels.length > 0) {
            handleTest([allModels[0]]);
        }
    };

    return (
        <div className="space-y-4 max-h-[60vh] overflow-y-auto">
            <div className="rounded-xl border bg-muted/20 p-3 space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                    <div>
                        <div className="text-sm font-medium text-card-foreground">{t('upstreamUpdateTitle')}</div>
                        <div className="text-xs text-muted-foreground">{t('upstreamUpdateDescription')}</div>
                    </div>
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={detectUpstreamUpdates.isPending}
                        onClick={handleDetectUpstream}
                        className="h-8"
                    >
                        {detectUpstreamUpdates.isPending ? (
                            <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                        ) : (
                            <RefreshCw className="h-4 w-4 mr-1" />
                        )}
                        {t('upstreamDetect')}
                    </Button>
                </div>
                {(pendingAddModels.length > 0 || pendingRemoveModels.length > 0) && (
                    <div className="space-y-2 text-xs">
                        {pendingAddModels.length > 0 && (
                            <div className="flex flex-wrap items-center gap-2">
                                <Badge variant="secondary" className="bg-green-500/15 text-green-700 dark:text-green-400">
                                    {t('upstreamPendingAdd', { count: pendingAddModels.length })}
                                </Badge>
                                <span className="text-muted-foreground break-all">{pendingAddModels.slice(0, 8).join(', ')}</span>
                                <Button size="sm" className="h-7" disabled={applyUpstreamUpdates.isPending} onClick={() => handleApplyUpstream('add')}>
                                    {t('upstreamApplyAdd')}
                                </Button>
                            </div>
                        )}
                        {pendingRemoveModels.length > 0 && (
                            <div className="flex flex-wrap items-center gap-2">
                                <Badge variant="secondary" className="bg-orange-500/15 text-orange-700 dark:text-orange-400">
                                    {t('upstreamPendingRemove', { count: pendingRemoveModels.length })}
                                </Badge>
                                <span className="text-muted-foreground break-all">{pendingRemoveModels.slice(0, 8).join(', ')}</span>
                                <Button size="sm" variant="outline" className="h-7" disabled={applyUpstreamUpdates.isPending} onClick={() => handleApplyUpstream('remove')}>
                                    {t('upstreamApplyRemove')}
                                </Button>
                            </div>
                        )}
                    </div>
                )}
            </div>

            {allModels.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                    {t('noModels')}
                </div>
            ) : (
                <>
                    {/* 工具栏 */}
                    <div className="flex items-center justify-between gap-3">
                        <Button
                            onClick={handleToggleAll}
                            variant="ghost"
                            size="sm"
                            className="h-8"
                        >
                            {selectedModels.size === allModels.length ? t('deselectAll') : t('selectAll')}
                        </Button>
                        <div className="flex flex-wrap items-center justify-end gap-2">
                            {selectedModels.size > 0 && (
                                <span className="text-xs text-muted-foreground">
                                    {t('selectedCount', { count: selectedModels.size })}
                                </span>
                            )}
                            <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
                                <Checkbox
                                    checked={dryRun}
                                    onChange={setDryRun}
                                    ariaLabel={t('dryRun')}
                                />
                                {t('dryRun')}
                            </label>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={handleDetectModelProtocols}
                                disabled={detectModelProtocols.isPending}
                                size="sm"
                                className="h-8"
                            >
                                {detectModelProtocols.isPending ? (
                                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                ) : (
                                    <Wand2 className="h-4 w-4 mr-1" />
                                )}
                                {detectModelProtocols.isPending ? t('modelProtocolDetecting') : t('modelProtocolDetect')}
                            </Button>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={handleApplyRecommendedProtocols}
                                disabled={applyModelProtocols.isPending || protocolResults.size === 0}
                                size="sm"
                                className="h-8"
                            >
                                {applyModelProtocols.isPending ? (
                                    <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                ) : (
                                    <CheckCircle2 className="h-4 w-4 mr-1" />
                                )}
                                {t('modelProtocolApplyRecommended')}
                            </Button>
                            <Button
                                onClick={() => selectedModels.size > 0 ? handleTest() : handleTestFirst()}
                                disabled={isTesting}
                                size="sm"
                                className="h-8"
                            >
                                {isTesting ? (
                                    <>
                                        <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                        {t('testing')}
                                    </>
                                ) : selectedModels.size > 0 ? (
                                    t('testSelected')
                                ) : (
                                    t('test')
                                )}
                            </Button>
                        </div>
                    </div>

                    {/* 模型列表 */}
                    <div className="space-y-2">
                        {allModels.map((model) => {
                            const isSelected = selectedModels.has(model);
                            const result = testResults.get(model);
                            const protocolResult = protocolResults.get(model);
                            const passedProtocolCount = protocolResult?.results.filter((item) => item.passed).length ?? 0;
                            const totalProtocolCount = protocolResult?.results.length ?? 0;

                            return (
                                <div
                                    key={model}
                                    className="flex items-center gap-3 p-3 border rounded-xl hover:bg-accent/5 transition-colors"
                                >
                                    <Checkbox
                                        id={`model-${model}`}
                                        ariaLabel={model}
                                        checked={isSelected}
                                        onChange={(checked) => {
                                            const newSet = new Set(selectedModels);
                                            if (checked) newSet.add(model);
                                            else newSet.delete(model);
                                            setSelectedModels(newSet);
                                        }}
                                    />
                                    <label
                                        htmlFor={`model-${model}`}
                                        className="flex-1 font-mono text-sm cursor-pointer select-all"
                                    >
                                        {model}
                                    </label>
                                    <div className="w-52 shrink-0 space-y-1">
                                        <Select
                                            value={getModelProtocolValue(model)}
                                            disabled={updateChannel.isPending}
                                            onValueChange={(value) => handleModelProtocolChange(model, value)}
                                        >
                                            <SelectTrigger className="h-8 rounded-xl text-xs">
                                                <SelectValue placeholder={t('modelProtocolDefault')} />
                                            </SelectTrigger>
                                            <SelectContent className="rounded-xl">
                                                <SelectItem className="rounded-xl" value="default">
                                                    {t('modelProtocolDefault')}
                                                </SelectItem>
                                                {protocolOptions.map((item) => (
                                                    <SelectItem key={item.value} className="rounded-xl" value={item.value}>
                                                        {item.label}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        {protocolResult && (
                                            <div className="flex flex-wrap items-center gap-1 text-[11px] text-muted-foreground">
                                                <Badge variant="secondary" className="bg-blue-500/10 text-blue-700 dark:text-blue-400">
                                                    {t('modelProtocolRecommended')}: {getProtocolLabel(protocolResult.recommended)}
                                                </Badge>
                                                <span>{t('modelProtocolProbeSummary', { passed: passedProtocolCount, total: totalProtocolCount })}</span>
                                            </div>
                                        )}
                                    </div>
                                    {result && (
                                        <div className="flex items-center gap-2">
                                            {result.delay !== undefined && (
                                                <span className="text-xs text-muted-foreground">
                                                    {result.delay}ms
                                                </span>
                                            )}
                                            <Badge
                                                variant="secondary"
                                                className={result.passed ? 'bg-green-500/15 text-green-700 dark:text-green-400' : 'bg-red-500/15 text-red-700 dark:text-red-400'}
                                            >
                                                {result.passed ? (
                                                    <>
                                                        <CheckCircle2 className="h-3 w-3 mr-1" />
                                                        {t('testPassed')}
                                                    </>
                                                ) : (
                                                    <>
                                                        <XCircle className="h-3 w-3 mr-1" />
                                                        {t('testFailed')}
                                                    </>
                                                )}
                                            </Badge>
                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>

                    {/* 测试结果摘要 */}
                    {testResults.size > 0 && (
                        <div className="text-xs text-muted-foreground">
                            {t('testSummary', {
                                total: testResults.size,
                                passed: Array.from(testResults.values()).filter((r) => r.passed).length,
                            })}
                        </div>
                    )}
                </>
            )}
        </div>
    );
}

