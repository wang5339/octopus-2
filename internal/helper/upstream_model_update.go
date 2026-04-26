package helper

import (
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/xstrings"
	"github.com/dlclark/regexp2"
)

// NormalizeModelNames 清洗模型名并去重，保留首次出现顺序。
func NormalizeModelNames(models []string) []string {
	normalized := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, item := range models {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized
}

// ChannelModelNames 返回渠道内置模型列表，不包含 custom_model。
// custom_model 通常是人工别名或手动补充项，不应被上游模型巡检自动删除。
func ChannelModelNames(channel model.Channel) []string {
	return NormalizeModelNames(xstrings.SplitTrimCompact(",", channel.Model))
}

func MergeModelNames(base []string, appended []string) []string {
	merged := NormalizeModelNames(base)
	seen := make(map[string]struct{}, len(merged))
	for _, item := range merged {
		seen[item] = struct{}{}
	}
	for _, item := range NormalizeModelNames(appended) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}
	return merged
}

func SubtractModelNames(base []string, removed []string) []string {
	removeSet := make(map[string]struct{}, len(removed))
	for _, item := range NormalizeModelNames(removed) {
		removeSet[item] = struct{}{}
	}
	result := make([]string, 0, len(base))
	for _, item := range NormalizeModelNames(base) {
		if _, ok := removeSet[item]; ok {
			continue
		}
		result = append(result, item)
	}
	return result
}

func IntersectModelNames(base []string, allowed []string) []string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, item := range NormalizeModelNames(allowed) {
		allowedSet[item] = struct{}{}
	}
	result := make([]string, 0, len(base))
	for _, item := range NormalizeModelNames(base) {
		if _, ok := allowedSet[item]; ok {
			result = append(result, item)
		}
	}
	return result
}

func ignoredByRule(modelName string, ignoredRules []string) bool {
	for _, rule := range NormalizeModelNames(ignoredRules) {
		regexBody, ok := strings.CutPrefix(rule, "regex:")
		if !ok {
			if rule == modelName {
				return true
			}
			continue
		}
		re, err := regexp2.Compile(strings.TrimSpace(regexBody), regexp2.ECMAScript)
		if err != nil {
			continue
		}
		matched, err := re.MatchString(modelName)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// CollectPendingUpstreamModelChanges 根据本地模型与上游模型计算待新增和待删除模型。
// 设计参考 new-api：
//   - 上游有、本地没有 => 待新增；
//   - 本地有、上游没有 => 待删除；
//   - 忽略列表支持精确匹配与 regex: 前缀正则；
//   - custom_model 不参与自动删除，避免误删人工别名。
func CollectPendingUpstreamModelChanges(channel model.Channel, upstreamModels []string) (pendingAddModels []string, pendingRemoveModels []string) {
	localModels := ChannelModelNames(channel)
	upstreamModels = NormalizeModelNames(upstreamModels)

	localSet := make(map[string]struct{}, len(localModels))
	for _, item := range localModels {
		localSet[item] = struct{}{}
	}
	upstreamSet := make(map[string]struct{}, len(upstreamModels))
	for _, item := range upstreamModels {
		upstreamSet[item] = struct{}{}
	}

	pendingAdd := make([]string, 0)
	for _, item := range upstreamModels {
		if _, ok := localSet[item]; ok {
			continue
		}
		if ignoredByRule(item, channel.UpstreamModelUpdateIgnoredModels) {
			continue
		}
		pendingAdd = append(pendingAdd, item)
	}

	pendingRemove := make([]string, 0)
	for _, item := range localModels {
		if _, ok := upstreamSet[item]; !ok {
			pendingRemove = append(pendingRemove, item)
		}
	}

	return NormalizeModelNames(pendingAdd), NormalizeModelNames(pendingRemove)
}
