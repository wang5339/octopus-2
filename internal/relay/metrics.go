package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/log"
)

const (
	logOmittedImageData     = "[image data omitted for storage]"
	logOmittedAudioData     = "[audio data omitted for storage]"
	logOmittedFileData      = "[file data omitted for storage]"
	logOmittedEmbeddingData = "[embedding data omitted for storage]"
)

// RelayMetrics 负责最终的日志收集与持久化
type RelayMetrics struct {
	APIKeyID     int
	RequestModel string
	StartTime    time.Time

	// 首 Token 时间
	FirstTokenTime time.Time

	// 请求和响应内容
	InternalRequest  *transformerModel.InternalLLMRequest
	InternalResponse *transformerModel.InternalLLMResponse

	// 统计指标
	ActualModel string
	Stats       model.StatsMetrics
}

func NewRelayMetrics(apiKeyID int, requestModel string, req *transformerModel.InternalLLMRequest) *RelayMetrics {
	return &RelayMetrics{
		APIKeyID:        apiKeyID,
		RequestModel:    requestModel,
		StartTime:       time.Now(),
		InternalRequest: req,
	}
}

func (m *RelayMetrics) SetFirstTokenTime(t time.Time) {
	m.FirstTokenTime = t
}

func (m *RelayMetrics) SetInternalResponse(resp *transformerModel.InternalLLMResponse, actualModel string) {
	m.InternalResponse = resp
	m.ActualModel = actualModel

	if resp == nil || resp.Usage == nil {
		return
	}

	usage := resp.Usage
	m.Stats.InputToken = usage.PromptTokens
	m.Stats.OutputToken = usage.CompletionTokens

	modelPrice := price.GetLLMPrice(actualModel)
	if modelPrice == nil {
		return
	}
	if usage.PromptTokensDetails == nil {
		usage.PromptTokensDetails = &transformerModel.PromptTokensDetails{
			CachedTokens: 0,
		}
	}
	if usage.AnthropicUsage {
		m.Stats.InputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead +
			float64(usage.PromptTokens)*modelPrice.Input +
			float64(usage.CacheCreationInputTokens)*modelPrice.CacheWrite) * 1e-6
	} else {
		m.Stats.InputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead + float64(usage.PromptTokens-usage.PromptTokensDetails.CachedTokens)*modelPrice.Input) * 1e-6
	}
	m.Stats.OutputCost = float64(usage.CompletionTokens) * modelPrice.Output * 1e-6
}

func (m *RelayMetrics) Save(ctx context.Context, success bool, err error, attempts []model.ChannelAttempt) {
	duration := time.Since(m.StartTime)

	globalStats := model.StatsMetrics{
		WaitTime:    duration.Milliseconds(),
		InputToken:  m.Stats.InputToken,
		OutputToken: m.Stats.OutputToken,
		InputCost:   m.Stats.InputCost,
		OutputCost:  m.Stats.OutputCost,
	}
	if success {
		globalStats.RequestSuccess = 1
	} else {
		globalStats.RequestFailed = 1
	}

	channelID, channelName := finalChannel(attempts)
	op.StatsTotalUpdate(globalStats)
	op.StatsHourlyUpdate(globalStats)
	op.StatsDailyUpdate(context.Background(), globalStats)
	op.StatsAPIKeyUpdate(m.APIKeyID, globalStats)
	op.StatsChannelUpdate(channelID, globalStats)
	op.StatsGroupUpdate(m.RequestModel, globalStats)

	log.Infof("relay complete: model=%s, channel=%d(%s), success=%t, duration=%dms, input_token=%d, output_token=%d, input_cost=%f, output_cost=%f, total_cost=%f, attempts=%d",
		m.RequestModel, channelID, channelName, success, duration.Milliseconds(),
		m.Stats.InputToken, m.Stats.OutputToken,
		m.Stats.InputCost, m.Stats.OutputCost, m.Stats.InputCost+m.Stats.OutputCost,
		len(attempts))

	m.saveLog(ctx, err, duration, attempts, channelID, channelName)
}

func finalChannel(attempts []model.ChannelAttempt) (int, string) {
	var lastID int
	var lastName string
	for i := len(attempts) - 1; i >= 0; i-- {
		a := attempts[i]
		if a.Status == model.AttemptSuccess {
			return a.ChannelID, a.ChannelName
		}
		if a.Status == model.AttemptFailed && lastID == 0 {
			lastID = a.ChannelID
			lastName = a.ChannelName
		}
	}
	return lastID, lastName
}

func (m *RelayMetrics) saveLog(ctx context.Context, err error, duration time.Duration, attempts []model.ChannelAttempt, channelID int, channelName string) {
	actualModel := m.ActualModel
	if actualModel == "" {
		actualModel = m.RequestModel
	}

	relayLog := model.RelayLog{
		Time:             m.StartTime.Unix(),
		RequestModelName: m.RequestModel,
		ChannelName:      channelName,
		ChannelId:        channelID,
		ActualModelName:  actualModel,
		UseTime:          int(duration.Milliseconds()),
		Attempts:         attempts,
		TotalAttempts:    len(attempts),
	}

	if apiKey, getErr := op.APIKeyGet(m.APIKeyID, ctx); getErr == nil {
		relayLog.RequestAPIKeyName = apiKey.Name
	}

	// 首字时间
	if !m.FirstTokenTime.IsZero() {
		relayLog.Ftut = int(m.FirstTokenTime.Sub(m.StartTime).Milliseconds())
	}

	// Usage
	if m.InternalResponse != nil && m.InternalResponse.Usage != nil {
		relayLog.InputTokens = int(m.InternalResponse.Usage.PromptTokens)
		relayLog.OutputTokens = int(m.InternalResponse.Usage.CompletionTokens)
		relayLog.Cost = m.Stats.InputCost + m.Stats.OutputCost
	}

	// 请求内容
	if m.InternalRequest != nil {
		reqForLog := m.filterRequestForLog(m.InternalRequest)
		if reqJSON, jsonErr := json.Marshal(reqForLog); jsonErr == nil {
			relayLog.RequestContent = string(reqJSON)
		}
	}

	// 响应内容
	if m.InternalResponse != nil {
		respForLog := m.filterResponseForLog(m.InternalResponse)
		if respJSON, jsonErr := json.Marshal(respForLog); jsonErr == nil {
			if m.InternalResponse.Usage != nil && m.InternalResponse.Usage.AnthropicUsage {
				respStr := string(respJSON)
				old := `"usage":{`
				insert := fmt.Sprintf(`"usage":{"cache_creation_input_tokens":%d,`, m.InternalResponse.Usage.CacheCreationInputTokens)
				respJSON = []byte(strings.Replace(respStr, old, insert, 1))
			}
			relayLog.ResponseContent = string(respJSON)
		}
	}

	// 错误信息
	if err != nil {
		relayLog.Error = err.Error()
	}

	if logErr := op.RelayLogAdd(ctx, relayLog); logErr != nil {
		log.Warnf("failed to save relay log: %v", logErr)
	}
}

// filterRequestForLog 创建请求的浅拷贝，过滤掉多模态大字段，避免图片/音频/文件
// base64 原文进入 RelayLog.RequestContent 造成数据库与实时日志缓存膨胀。
func (m *RelayMetrics) filterRequestForLog(req *transformerModel.InternalLLMRequest) *transformerModel.InternalLLMRequest {
	if req == nil {
		return nil
	}

	filtered := *req
	if len(req.Messages) > 0 {
		filtered.Messages = make([]transformerModel.Message, len(req.Messages))
		for i := range req.Messages {
			filtered.Messages[i] = *filterMessageForLog(&req.Messages[i])
		}
	}
	return &filtered
}

// filterResponseForLog 创建响应的浅拷贝，过滤掉多模态大字段，减少日志存储压力。
func (m *RelayMetrics) filterResponseForLog(resp *transformerModel.InternalLLMResponse) *transformerModel.InternalLLMResponse {
	if resp == nil {
		return nil
	}

	filtered := *resp
	if len(resp.Choices) > 0 {
		filtered.Choices = make([]transformerModel.Choice, len(resp.Choices))
		for i, choice := range resp.Choices {
			filtered.Choices[i] = choice
			filtered.Choices[i].Message = filterMessageForLog(choice.Message)
			filtered.Choices[i].Delta = filterMessageForLog(choice.Delta)
		}
	}
	if len(resp.ImageData) > 0 {
		filtered.ImageData = make([]transformerModel.ImageObject, len(resp.ImageData))
		for i, image := range resp.ImageData {
			filtered.ImageData[i] = image
			if image.B64JSON != nil && *image.B64JSON != "" {
				omitted := logOmittedImageData
				filtered.ImageData[i].B64JSON = &omitted
			}
		}
	}
	if len(resp.EmbeddingData) > 0 {
		filtered.EmbeddingData = make([]transformerModel.EmbeddingObject, len(resp.EmbeddingData))
		for i, embedding := range resp.EmbeddingData {
			filtered.EmbeddingData[i] = embedding
			if len(embedding.Embedding.FloatArray) > 0 || embedding.Embedding.Base64String != nil {
				omitted := logOmittedEmbeddingData
				filtered.EmbeddingData[i].Embedding = transformerModel.Embedding{
					Base64String: &omitted,
				}
			}
		}
	}
	return &filtered
}

func filterMessageForLog(msg *transformerModel.Message) *transformerModel.Message {
	if msg == nil {
		return nil
	}

	filtered := *msg
	filtered.Images = nil

	if len(msg.Content.MultipleContent) > 0 {
		parts := make([]transformerModel.MessageContentPart, len(msg.Content.MultipleContent))
		for i, part := range msg.Content.MultipleContent {
			parts[i] = filterContentPartForLog(part)
		}
		filtered.Content = transformerModel.MessageContent{
			Content:         msg.Content.Content,
			MultipleContent: parts,
		}
	}

	if msg.Audio != nil && msg.Audio.Data != "" {
		audio := *msg.Audio
		audio.Data = logOmittedAudioData
		filtered.Audio = &audio
	}

	return &filtered
}

func filterContentPartForLog(part transformerModel.MessageContentPart) transformerModel.MessageContentPart {
	filtered := part

	if part.ImageURL != nil {
		imageURL := *part.ImageURL
		imageURL.URL = logOmittedImageData
		filtered.ImageURL = &imageURL
	}

	if part.Audio != nil && part.Audio.Data != "" {
		audio := *part.Audio
		audio.Data = logOmittedAudioData
		filtered.Audio = &audio
	}

	if part.File != nil && part.File.FileData != "" {
		file := *part.File
		file.FileData = logOmittedFileData
		filtered.File = &file
	}

	return filtered
}
