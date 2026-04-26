package task

import (
	"context"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/utils/log"
)

var lastSyncModelsTime = time.Now()

// SyncModelsTask 同步模型任务
func SyncModelsTask() {
	log.Debugf("sync models task started")
	startTime := time.Now()
	defer func() {
		log.Debugf("sync models task finished, sync time: %s", time.Since(startTime))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	channels, err := op.ChannelList(ctx)
	if err != nil {
		log.Errorf("failed to list channels: %v", err)
		return
	}
	totalNewModels := make([]string, 0, 128)
	seenTotalNewModels := make(map[string]struct{}, 128)
	for _, channel := range channels {
		if !channel.AutoSync {
			continue
		}
		fetchModels, err := helper.FetchModels(ctx, channel)
		if err != nil {
			log.Warnf("failed to fetch models for channel %s: %v", channel.Name, err)
			continue
		}
		oldModels := helper.ChannelModelNames(channel)
		newModels := helper.NormalizeModelNames(fetchModels)
		for _, m := range newModels {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			m = strings.ToLower(m)
			if _, ok := seenTotalNewModels[m]; ok {
				continue
			}
			seenTotalNewModels[m] = struct{}{}
			totalNewModels = append(totalNewModels, m)
		}
		addedModels, removedModels := helper.CollectPendingUpstreamModelChanges(channel, newModels)
		now := time.Now().Unix()
		nextDetected := addedModels
		modelChanged := false
		if len(addedModels) > 0 {
			// AutoSync 只自动追加新增模型，不自动删除上游已消失模型。
			// 这样比全量覆盖更安全，人工别名和临时可用模型不会被定时任务误删。
			mergedModels := helper.MergeModelNames(oldModels, addedModels)
			if strings.Join(mergedModels, ",") != strings.Join(oldModels, ",") {
				modelChanged = true
				channel.Model = strings.Join(mergedModels, ",")
				nextDetected = []string{}
			}
		}

		updateReq := &model.ChannelUpdateRequest{
			ID:                                    channel.ID,
			UpstreamModelUpdateLastCheckTime:      &now,
			UpstreamModelUpdateLastDetectedModels: &nextDetected,
			UpstreamModelUpdateLastRemovedModels:  &removedModels,
		}
		if modelChanged {
			fetchModelStr := channel.Model
			updateReq.Model = &fetchModelStr
		}
		if _, err := op.ChannelUpdate(updateReq, ctx); err != nil {
			log.Errorf("failed to update channel %s: %v", channel.Name, err)
			continue
		}
		if len(removedModels) > 0 {
			log.Infof("channel %s upstream removed models detected, waiting for manual apply: %v", channel.Name, removedModels)
		}

		// 自动分组
		if modelChanged {
			helper.ChannelAutoGroup(&channel, ctx)
		}
	}
	llmPrice, err := op.LLMList(ctx)
	if err != nil {
		log.Errorf("failed to list models price: %v", err)
		return
	}
	llmPriceNames := make([]string, 0, len(llmPrice))
	for _, price := range llmPrice {
		llmPriceNames = append(llmPriceNames, price.Name)
	}

	deletedNorm, addedNorm := helper.SubtractModelNames(llmPriceNames, totalNewModels), helper.SubtractModelNames(totalNewModels, llmPriceNames)
	if len(deletedNorm) > 0 {
		if err := helper.LLMPriceDeleteFromDBWithNoPrice(deletedNorm, ctx); err != nil {
			log.Errorf("failed to batch delete models price: %v", err)
		}
	}
	if len(addedNorm) > 0 {
		if err := helper.LLMPriceAddToDB(addedNorm, ctx); err != nil {
			log.Errorf("failed to add models price: %v", err)
		}
	}
	lastSyncModelsTime = time.Now()
}

func GetLastSyncModelsTime() time.Time {
	return lastSyncModelsTime
}
