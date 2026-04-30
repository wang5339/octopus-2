package op

import (
	"context"
	"fmt"
	"strings"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
)

var llmModelCache = cache.New[string, model.LLMPrice](16)

func normalizeLLMName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func LLMList(ctx context.Context) ([]model.LLMInfo, error) {
	models := make([]model.LLMInfo, 0, llmModelCache.Len())
	for m, cost := range llmModelCache.GetAll() {
		models = append(models, model.LLMInfo{
			Name:     m,
			LLMPrice: cost,
		})
	}
	return models, nil
}

func LLMUpdate(model model.LLMInfo, ctx context.Context) error {
	model.Name = normalizeLLMName(model.Name)
	if model.Name == "" {
		return fmt.Errorf("model name is empty")
	}
	_, ok := llmModelCache.Get(model.Name)
	if !ok {
		return fmt.Errorf("model not found")
	}
	if err := db.GetDB().WithContext(ctx).Save(model).Error; err != nil {
		return err
	}
	llmModelCache.Set(model.Name, model.LLMPrice)
	return nil
}

func LLMDelete(modelName string, ctx context.Context) error {
	modelName = normalizeLLMName(modelName)
	if modelName == "" {
		return fmt.Errorf("model name is empty")
	}
	_, ok := llmModelCache.Get(modelName)
	if !ok {
		return fmt.Errorf("model not found")
	}
	if err := db.GetDB().WithContext(ctx).Delete(&model.LLMInfo{Name: modelName}).Error; err != nil {
		return err
	}
	llmModelCache.Del(modelName)
	return nil
}
func LLMBatchDelete(modelNames []string, ctx context.Context) error {
	if len(modelNames) == 0 {
		return nil
	}
	modelNames = normalizeLLMNames(modelNames)
	if len(modelNames) == 0 {
		return nil
	}
	if err := db.GetDB().WithContext(ctx).Where("name IN ?", modelNames).Delete(&model.LLMInfo{}).Error; err != nil {
		return err
	}
	llmModelCache.Del(modelNames...)
	return nil
}
func LLMCreate(model model.LLMInfo, ctx context.Context) error {
	model.Name = normalizeLLMName(model.Name)
	if model.Name == "" {
		return fmt.Errorf("model name is empty")
	}
	_, ok := llmModelCache.Get(model.Name)
	if ok {
		return fmt.Errorf("model already exists")
	}
	if err := db.GetDB().WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	llmModelCache.Set(model.Name, model.LLMPrice)
	return nil
}
func LLMBatchCreate(llmInfos []model.LLMInfo, ctx context.Context) error {
	if len(llmInfos) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(llmInfos))
	newLLMInfos := make([]model.LLMInfo, 0, len(llmInfos))
	for _, llmInfo := range llmInfos {
		llmInfo.Name = normalizeLLMName(llmInfo.Name)
		if llmInfo.Name == "" {
			continue
		}
		if _, ok := seen[llmInfo.Name]; ok {
			continue
		}
		if _, ok := llmModelCache.Get(llmInfo.Name); ok {
			continue
		}
		seen[llmInfo.Name] = struct{}{}
		newLLMInfos = append(newLLMInfos, llmInfo)
	}
	if len(newLLMInfos) == 0 {
		return nil
	}
	if err := db.GetDB().WithContext(ctx).Create(&newLLMInfos).Error; err != nil {
		return err
	}
	for _, llmInfo := range newLLMInfos {
		llmModelCache.Set(llmInfo.Name, llmInfo.LLMPrice)
	}
	return nil
}
func LLMGet(name string) (model.LLMPrice, error) {
	name = normalizeLLMName(name)
	if name == "" {
		return model.LLMPrice{}, fmt.Errorf("model name is empty")
	}
	price, ok := llmModelCache.Get(name)
	if !ok {
		return model.LLMPrice{}, fmt.Errorf("model not found")
	}
	return price, nil
}

func llmRefreshCache(ctx context.Context) error {
	models := []model.LLMInfo{}
	if err := db.GetDB().WithContext(ctx).Find(&models).Error; err != nil {
		return err
	}
	llmModelCache.Clear()
	for _, model := range models {
		model.Name = normalizeLLMName(model.Name)
		if model.Name == "" {
			continue
		}
		llmModelCache.Set(model.Name, model.LLMPrice)
	}
	return nil
}

func normalizeLLMNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		name = normalizeLLMName(name)
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
