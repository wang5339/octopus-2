package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/helper"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/server/middleware"
	"github.com/bestruirui/octopus/internal/server/resp"
	"github.com/bestruirui/octopus/internal/server/router"
	"github.com/bestruirui/octopus/internal/task"
	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
	transformerOutbound "github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/gin-gonic/gin"
)

func init() {
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(listChannel),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createChannel),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateChannel),
		).
		AddRoute(
			router.NewRoute("/enable", http.MethodPost).
				Handle(enableChannel),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteChannel),
		).
		AddRoute(
			router.NewRoute("/fetch-model", http.MethodPost).
				Handle(fetchModel),
		).
		AddRoute(
			router.NewRoute("/test-models", http.MethodPost).
				Handle(testChannelModels),
		).
		AddRoute(
			router.NewRoute("/test-models-by-config", http.MethodPost).
				Handle(testChannelModelsByConfig),
		).
		AddRoute(
			router.NewRoute("/upstream-updates/detect", http.MethodPost).
				Handle(detectChannelUpstreamUpdates),
		).
		AddRoute(
			router.NewRoute("/upstream-updates/apply", http.MethodPost).
				Handle(applyChannelUpstreamUpdates),
		).
		AddRoute(
			router.NewRoute("/model-protocols/detect", http.MethodPost).
				Handle(detectChannelModelProtocols),
		).
		AddRoute(
			router.NewRoute("/model-protocols/apply", http.MethodPost).
				Handle(applyChannelModelProtocols),
		)
	router.NewGroupRouter("/api/v1/channel").
		Use(middleware.Auth()).
		AddRoute(
			router.NewRoute("/sync", http.MethodPost).
				Handle(syncChannel),
		).
		AddRoute(
			router.NewRoute("/last-sync-time", http.MethodGet).
				Handle(getLastSyncTime),
		)
}

func listChannel(c *gin.Context) {
	channels, err := op.ChannelList(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	for i, channel := range channels {
		stats := op.StatsChannelGet(channel.ID)
		channels[i].Stats = &stats
	}
	resp.Success(c, channels)
}

func createChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.ChannelCreate(&channel, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
	go func(channel *model.Channel) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		modelStr := channel.Model + "," + channel.CustomModel
		modelArray := strings.Split(modelStr, ",")
		helper.LLMPriceAddToDB(modelArray, ctx)
		helper.ChannelBaseUrlDelayUpdate(channel, ctx)
		helper.ChannelAutoGroup(channel, ctx)
	}(&channel)
	resp.Success(c, channel)
}

func updateChannel(c *gin.Context) {
	var req model.ChannelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelUpdate(&req, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	stats := op.StatsChannelGet(channel.ID)
	channel.Stats = &stats
	go func(channel *model.Channel) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		modelStr := channel.Model + "," + channel.CustomModel
		modelArray := strings.Split(modelStr, ",")
		helper.LLMPriceAddToDB(modelArray, ctx)
		helper.ChannelBaseUrlDelayUpdate(channel, ctx)
		helper.ChannelAutoGroup(channel, ctx)
	}(channel)
	resp.Success(c, channel)
}

func enableChannel(c *gin.Context) {
	var request struct {
		ID      int  `json:"id"`
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	if err := op.ChannelEnabled(request.ID, request.Enabled, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}

func deleteChannel(c *gin.Context) {
	id := c.Param("id")
	idNum, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidParam)
		return
	}
	if err := op.ChannelDel(idNum, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, nil)
}
func fetchModel(c *gin.Context) {
	var request model.Channel
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	models, err := helper.FetchModels(c.Request.Context(), request)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, models)
}

func resolveModelOutboundType(channel model.Channel, modelName string) transformerOutbound.OutboundType {
	trimmedModel := strings.TrimSpace(modelName)
	for _, item := range channel.ModelProtocolOverrides {
		if strings.EqualFold(strings.TrimSpace(item.Model), trimmedModel) {
			return item.Type
		}
	}
	return channel.Type
}

type modelProtocolDetectRequest struct {
	ID     int                                `json:"id" binding:"required"`
	Models []string                           `json:"models"`
	Types  []transformerOutbound.OutboundType `json:"types"`
}

type modelProtocolProbeResult struct {
	Type   transformerOutbound.OutboundType `json:"type"`
	Passed bool                             `json:"passed"`
	Error  string                           `json:"error,omitempty"`
	Delay  int                              `json:"delay,omitempty"`
}

type modelProtocolDetectResult struct {
	Model       string                            `json:"model"`
	Recommended *transformerOutbound.OutboundType `json:"recommended,omitempty"`
	Results     []modelProtocolProbeResult        `json:"results"`
}

type modelProtocolApplyRequest struct {
	ID        int                           `json:"id" binding:"required"`
	Overrides []model.ModelProtocolOverride `json:"overrides"`
}

type upstreamModelUpdateRequest struct {
	ID           int      `json:"id" binding:"required"`
	AddModels    []string `json:"add_models"`
	RemoveModels []string `json:"remove_models"`
	IgnoreModels []string `json:"ignore_models"`
}

type upstreamModelUpdateResult struct {
	ChannelID     int      `json:"channel_id"`
	ChannelName   string   `json:"channel_name"`
	AddModels     []string `json:"add_models"`
	RemoveModels  []string `json:"remove_models"`
	LastCheckTime int64    `json:"last_check_time"`
}

func detectChannelUpstreamUpdates(c *gin.Context) {
	var req upstreamModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelGet(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	upstreamModels, err := helper.FetchModels(c.Request.Context(), *channel)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	addModels, removeModels := helper.CollectPendingUpstreamModelChanges(*channel, upstreamModels)
	now := time.Now().Unix()
	if _, err := op.ChannelUpdate(&model.ChannelUpdateRequest{
		ID:                                    channel.ID,
		UpstreamModelUpdateLastCheckTime:      &now,
		UpstreamModelUpdateLastDetectedModels: &addModels,
		UpstreamModelUpdateLastRemovedModels:  &removeModels,
	}, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, upstreamModelUpdateResult{
		ChannelID:     channel.ID,
		ChannelName:   channel.Name,
		AddModels:     addModels,
		RemoveModels:  removeModels,
		LastCheckTime: now,
	})
}

func applyChannelUpstreamUpdates(c *gin.Context) {
	var req upstreamModelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelGet(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	pendingAddModels := helper.NormalizeModelNames(channel.UpstreamModelUpdateLastDetectedModels)
	pendingRemoveModels := helper.NormalizeModelNames(channel.UpstreamModelUpdateLastRemovedModels)
	addModels := helper.IntersectModelNames(req.AddModels, pendingAddModels)
	ignoreModels := helper.IntersectModelNames(req.IgnoreModels, pendingAddModels)
	removeModels := helper.IntersectModelNames(req.RemoveModels, pendingRemoveModels)
	removeModels = helper.SubtractModelNames(removeModels, addModels)

	originModels := helper.ChannelModelNames(*channel)
	nextModels := helper.SubtractModelNames(helper.MergeModelNames(originModels, addModels), removeModels)
	nextModelStr := strings.Join(nextModels, ",")
	remainingAddModels := helper.SubtractModelNames(pendingAddModels, append(addModels, ignoreModels...))
	remainingRemoveModels := helper.SubtractModelNames(pendingRemoveModels, removeModels)
	ignoredModels := helper.MergeModelNames(channel.UpstreamModelUpdateIgnoredModels, ignoreModels)
	if len(addModels) > 0 {
		ignoredModels = helper.SubtractModelNames(ignoredModels, addModels)
	}
	now := time.Now().Unix()

	if _, err := op.ChannelUpdate(&model.ChannelUpdateRequest{
		ID:                                    channel.ID,
		Model:                                 &nextModelStr,
		UpstreamModelUpdateLastCheckTime:      &now,
		UpstreamModelUpdateLastDetectedModels: &remainingAddModels,
		UpstreamModelUpdateLastRemovedModels:  &remainingRemoveModels,
		UpstreamModelUpdateIgnoredModels:      &ignoredModels,
	}, c.Request.Context()); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if len(removeModels) > 0 {
		keys := make([]model.GroupIDAndLLMName, len(removeModels))
		for i, modelName := range removeModels {
			keys[i] = model.GroupIDAndLLMName{ChannelID: channel.ID, ModelName: modelName}
		}
		if err := op.GroupItemBatchDelByChannelAndModels(keys, c.Request.Context()); err != nil {
			resp.Error(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if len(addModels) > 0 {
		helper.LLMPriceAddToDB(addModels, c.Request.Context())
		channel.Model = nextModelStr
		helper.ChannelAutoGroup(channel, c.Request.Context())
	}

	resp.Success(c, map[string]any{
		"id":                      channel.ID,
		"added_models":            addModels,
		"removed_models":          removeModels,
		"ignored_models":          ignoreModels,
		"remaining_models":        remainingAddModels,
		"remaining_remove_models": remainingRemoveModels,
		"models":                  nextModelStr,
	})
}

func syncChannel(c *gin.Context) {
	task.SyncModelsTask()
	resp.Success(c, nil)
}

func getLastSyncTime(c *gin.Context) {
	time := task.GetLastSyncModelsTime()
	resp.Success(c, time)
}

func testChannelModels(c *gin.Context) {
	type TestModelRequest struct {
		ChannelID int      `json:"channel_id"`
		Models    []string `json:"models"`
	}

	var req TestModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	if len(req.Models) == 0 {
		resp.Error(c, http.StatusBadRequest, "models list is empty")
		return
	}

	// 限制模型数量，防止资源耗尽
	if len(req.Models) > 50 {
		resp.Error(c, http.StatusBadRequest, "too many models (max 50)")
		return
	}

	channel, err := op.ChannelGet(req.ChannelID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	results := make([]testModelResult, 0, len(req.Models))

	for _, modelName := range req.Models {
		results = append(results, runChannelModelTest(c, channel, channel.Type, modelName))
	}

	c.JSON(http.StatusOK, results)
}

// testModelResult 单次模型测试结果（handler 内部用）
type testModelResult struct {
	Model  string `json:"model"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
	Delay  int    `json:"delay,omitempty"`
}

// runChannelModelTest 对单个模型执行一次完整测试。
// 默认尊重 model_protocol_overrides，适合已保存渠道的常规测试。
func runChannelModelTest(c *gin.Context, channel *model.Channel, channelType transformerOutbound.OutboundType, modelName string) testModelResult {
	return runChannelModelTestWithType(c, channel, resolveModelOutboundType(*channel, modelName), modelName)
}

// runChannelModelTestWithType 使用指定协议测试模型。
// 协议探测和表单临时配置测试必须走这里，避免被已有模型级覆盖干扰。
func runChannelModelTestWithType(c *gin.Context, channel *model.Channel, channelType transformerOutbound.OutboundType, modelName string) testModelResult {
	result := testModelResult{Model: modelName}

	httpClient, err := helper.ChannelHttpClient(channel)
	if err != nil {
		result.Error = "Failed to create HTTP client: " + err.Error()
		return result
	}

	baseURL := channel.GetBaseUrl()
	delay, err := helper.GetUrlDelay(httpClient, baseURL, c.Request.Context())
	if err != nil {
		result.Error = "Connectivity test failed: " + err.Error()
		return result
	}
	result.Delay = delay

	content := "1+1=?"
	maxTokens := int64(1)
	temperature := 0.0
	testReq := transformerModel.InternalLLMRequest{
		Model:       modelName,
		Messages:    []transformerModel.Message{{Role: "user", Content: transformerModel.MessageContent{Content: &content}}},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	channelKey := channel.GetChannelKey()
	if channelKey.ChannelKey == "" {
		result.Error = "No available API key"
		return result
	}

	outboundAdapter := transformerOutbound.GetForModel(channelType, modelName)
	if outboundAdapter == nil {
		result.Error = "Unsupported channel type"
		return result
	}

	outboundReq, err := outboundAdapter.TransformRequest(c.Request.Context(), &testReq, baseURL, channelKey.ChannelKey)
	if err != nil {
		result.Error = "Failed to build request: " + err.Error()
		return result
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	httpResp, err := httpClient.Do(outboundReq.WithContext(ctx))
	if err != nil {
		result.Error = "LLM request failed: " + err.Error()
		return result
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
		result.Passed = true
	} else if httpResp.StatusCode == http.StatusTooManyRequests {
		result.Passed = true
		result.Error = "Rate limited (429), but channel is reachable"
	} else {
		result.Error = "LLM returned status: " + httpResp.Status
	}

	return result
}

func testChannelModelsByConfig(c *gin.Context) {
	type TestModelByConfigRequest struct {
		Type     transformerOutbound.OutboundType `json:"type"`
		BaseUrls []model.BaseUrl                  `json:"base_urls"`
		Keys     []struct {
			Enabled    bool   `json:"enabled"`
			ChannelKey string `json:"channel_key"`
		} `json:"keys"`
		Proxy        bool                 `json:"proxy"`
		ChannelProxy *string              `json:"channel_proxy"`
		CustomHeader []model.CustomHeader `json:"custom_header"`
		Models       []string             `json:"models"`
	}

	var req TestModelByConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}

	if len(req.Models) == 0 {
		resp.Error(c, http.StatusBadRequest, "models list is empty")
		return
	}

	// 限制模型数量，防止资源耗尽
	if len(req.Models) > 50 {
		resp.Error(c, http.StatusBadRequest, "too many models (max 50)")
		return
	}

	channel := &model.Channel{
		Type:         req.Type,
		BaseUrls:     req.BaseUrls,
		Proxy:        req.Proxy,
		ChannelProxy: req.ChannelProxy,
		CustomHeader: req.CustomHeader,
	}
	for _, k := range req.Keys {
		channel.Keys = append(channel.Keys, model.ChannelKey{
			Enabled:    k.Enabled,
			ChannelKey: k.ChannelKey,
		})
	}

	results := make([]testModelResult, 0, len(req.Models))

	for _, modelName := range req.Models {
		results = append(results, runChannelModelTestWithType(c, channel, req.Type, modelName))
	}

	c.JSON(http.StatusOK, results)
}

func detectChannelModelProtocols(c *gin.Context) {
	var req modelProtocolDetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelGet(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	models := helper.NormalizeModelNames(req.Models)
	if len(models) == 0 {
		models = helper.ChannelModelNames(*channel)
	}
	if len(models) == 0 {
		resp.Error(c, http.StatusBadRequest, "models list is empty")
		return
	}
	if len(models) > 50 {
		resp.Error(c, http.StatusBadRequest, "too many models (max 50)")
		return
	}

	types := filterModelProtocolTypes(req.Types)
	results := make([]modelProtocolDetectResult, 0, len(models))
	for _, modelName := range models {
		itemResults := make([]modelProtocolProbeResult, 0, len(types))
		for _, protocolType := range types {
			testResult := runChannelModelTestWithType(c, channel, protocolType, modelName)
			itemResults = append(itemResults, modelProtocolProbeResult{
				Type:   protocolType,
				Passed: testResult.Passed,
				Error:  testResult.Error,
				Delay:  testResult.Delay,
			})
		}
		results = append(results, modelProtocolDetectResult{
			Model:       modelName,
			Recommended: recommendModelProtocol(channel.Type, modelName, itemResults),
			Results:     itemResults,
		})
	}

	resp.Success(c, results)
}

func applyChannelModelProtocols(c *gin.Context) {
	var req modelProtocolApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, resp.ErrInvalidJSON)
		return
	}
	channel, err := op.ChannelGet(req.ID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	knownModels := make(map[string]struct{})
	for _, modelName := range helper.ChannelModelNames(*channel) {
		knownModels[strings.ToLower(strings.TrimSpace(modelName))] = struct{}{}
	}

	touchedModels := make(map[string]struct{})
	incoming := make(map[string]model.ModelProtocolOverride)
	for _, override := range req.Overrides {
		modelName := strings.TrimSpace(override.Model)
		key := strings.ToLower(modelName)
		if modelName == "" || !isDetectableModelProtocolType(override.Type) {
			continue
		}
		if len(knownModels) > 0 {
			if _, ok := knownModels[key]; !ok {
				continue
			}
		}
		touchedModels[key] = struct{}{}
		if override.Type == channel.Type {
			continue
		}
		incoming[key] = model.ModelProtocolOverride{Model: modelName, Type: override.Type}
	}

	nextOverrides := make([]model.ModelProtocolOverride, 0, len(channel.ModelProtocolOverrides)+len(incoming))
	for _, override := range channel.ModelProtocolOverrides {
		modelName := strings.TrimSpace(override.Model)
		key := strings.ToLower(modelName)
		if modelName == "" || !isDetectableModelProtocolType(override.Type) {
			continue
		}
		if _, touched := touchedModels[key]; touched {
			continue
		}
		if len(knownModels) > 0 {
			if _, ok := knownModels[key]; !ok {
				continue
			}
		}
		nextOverrides = append(nextOverrides, model.ModelProtocolOverride{Model: modelName, Type: override.Type})
	}
	for _, modelName := range helper.ChannelModelNames(*channel) {
		key := strings.ToLower(strings.TrimSpace(modelName))
		if override, ok := incoming[key]; ok {
			nextOverrides = append(nextOverrides, override)
			delete(incoming, key)
		}
	}
	for _, override := range incoming {
		nextOverrides = append(nextOverrides, override)
	}

	updatedChannel, err := op.ChannelUpdate(&model.ChannelUpdateRequest{
		ID:                     channel.ID,
		ModelProtocolOverrides: &nextOverrides,
	}, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, map[string]any{
		"id":                       updatedChannel.ID,
		"applied":                  nextOverrides,
		"model_protocol_overrides": updatedChannel.ModelProtocolOverrides,
	})
}

func filterModelProtocolTypes(types []transformerOutbound.OutboundType) []transformerOutbound.OutboundType {
	if len(types) == 0 {
		return candidateModelProtocolTypes()
	}
	seen := make(map[transformerOutbound.OutboundType]struct{})
	result := make([]transformerOutbound.OutboundType, 0, len(types))
	for _, protocolType := range types {
		if !isDetectableModelProtocolType(protocolType) {
			continue
		}
		if _, ok := seen[protocolType]; ok {
			continue
		}
		seen[protocolType] = struct{}{}
		result = append(result, protocolType)
	}
	if len(result) == 0 {
		return candidateModelProtocolTypes()
	}
	return result
}

func candidateModelProtocolTypes() []transformerOutbound.OutboundType {
	return []transformerOutbound.OutboundType{
		transformerOutbound.OutboundTypeOpenAIChat,
		transformerOutbound.OutboundTypeOpenAIResponse,
		transformerOutbound.OutboundTypeAnthropic,
		transformerOutbound.OutboundTypeGemini,
		transformerOutbound.OutboundTypeVolcengine,
	}
}

func isDetectableModelProtocolType(protocolType transformerOutbound.OutboundType) bool {
	switch protocolType {
	case transformerOutbound.OutboundTypeOpenAIChat,
		transformerOutbound.OutboundTypeOpenAIResponse,
		transformerOutbound.OutboundTypeAnthropic,
		transformerOutbound.OutboundTypeGemini,
		transformerOutbound.OutboundTypeVolcengine:
		return true
	default:
		return false
	}
}

func recommendModelProtocol(channelType transformerOutbound.OutboundType, modelName string, results []modelProtocolProbeResult) *transformerOutbound.OutboundType {
	passed := make(map[transformerOutbound.OutboundType]struct{})
	for _, result := range results {
		if result.Passed {
			passed[result.Type] = struct{}{}
		}
	}
	if len(passed) == 0 {
		return nil
	}
	if _, ok := passed[channelType]; ok && isDetectableModelProtocolType(channelType) {
		return outboundTypePtr(channelType)
	}

	lowerModel := strings.ToLower(strings.TrimSpace(modelName))
	if strings.HasPrefix(lowerModel, "claude-") || strings.Contains(lowerModel, "claude") {
		if _, ok := passed[transformerOutbound.OutboundTypeAnthropic]; ok {
			return outboundTypePtr(transformerOutbound.OutboundTypeAnthropic)
		}
	}
	if strings.HasPrefix(lowerModel, "gemini-") || strings.Contains(lowerModel, "gemini") {
		if _, ok := passed[transformerOutbound.OutboundTypeGemini]; ok {
			return outboundTypePtr(transformerOutbound.OutboundTypeGemini)
		}
	}
	if strings.HasPrefix(lowerModel, "gpt-5") || strings.HasPrefix(lowerModel, "o1") || strings.HasPrefix(lowerModel, "o3") || strings.HasPrefix(lowerModel, "o4") || strings.HasPrefix(lowerModel, "o5") {
		if _, ok := passed[transformerOutbound.OutboundTypeOpenAIResponse]; ok {
			return outboundTypePtr(transformerOutbound.OutboundTypeOpenAIResponse)
		}
	}

	for _, protocolType := range candidateModelProtocolTypes() {
		if _, ok := passed[protocolType]; ok {
			return outboundTypePtr(protocolType)
		}
	}
	return nil
}

func outboundTypePtr(protocolType transformerOutbound.OutboundType) *transformerOutbound.OutboundType {
	return &protocolType
}
