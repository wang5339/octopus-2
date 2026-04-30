package outbound

import (
	"strings"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound/authropic"
	"github.com/bestruirui/octopus/internal/transformer/outbound/gemini"
	"github.com/bestruirui/octopus/internal/transformer/outbound/openai"
	"github.com/bestruirui/octopus/internal/transformer/outbound/volcengine"
)

type OutboundType int

const (
	OutboundTypeOpenAIChat            OutboundType = 0
	OutboundTypeOpenAIResponse        OutboundType = 1
	OutboundTypeAnthropic             OutboundType = 2
	OutboundTypeGemini                OutboundType = 3
	OutboundTypeVolcengine            OutboundType = 4
	OutboundTypeOpenAIEmbedding       OutboundType = 5
	OutboundTypeOpenAIImageGeneration OutboundType = 6
	// 7/8 是已移除渠道的历史编号。保留 Zen=9 的显式编号，
	// 避免历史数据库里的 Zen 渠道被错位解析。
	OutboundTypeZen OutboundType = 9
)

// EmbeddingChannelTypes 定义支持 embedding 请求的 channel 类型集合
var EmbeddingChannelTypes = map[OutboundType]bool{
	OutboundTypeOpenAIEmbedding: true,
}

// ImageGenerationChannelTypes 定义支持 image generation 请求的 channel 类型集合
var ImageGenerationChannelTypes = map[OutboundType]bool{
	OutboundTypeOpenAIImageGeneration: true,
}

// ChatChannelTypes 定义支持 chat 请求的 channel 类型集合
var ChatChannelTypes = map[OutboundType]bool{
	OutboundTypeOpenAIChat:     true,
	OutboundTypeOpenAIResponse: true,
	OutboundTypeAnthropic:      true,
	OutboundTypeGemini:         true,
	OutboundTypeVolcengine:     true,
	OutboundTypeZen:            true,
}

// IsEmbeddingChannelType 判断 channel 类型是否支持 embedding 请求
func IsEmbeddingChannelType(channelType OutboundType) bool {
	return EmbeddingChannelTypes[channelType]
}

// IsImageGenerationChannelType 判断 channel 类型是否支持 image generation 请求
func IsImageGenerationChannelType(channelType OutboundType) bool {
	return ImageGenerationChannelTypes[channelType]
}

// IsChatChannelType 判断 channel 类型是否支持 chat 请求
func IsChatChannelType(channelType OutboundType) bool {
	return ChatChannelTypes[channelType]
}

var outboundFactories = map[OutboundType]func() model.Outbound{
	OutboundTypeOpenAIChat:            func() model.Outbound { return &openai.ChatOutbound{} },
	OutboundTypeOpenAIResponse:        func() model.Outbound { return &openai.ResponseOutbound{} },
	OutboundTypeOpenAIEmbedding:       func() model.Outbound { return &openai.EmbeddingOutbound{} },
	OutboundTypeOpenAIImageGeneration: func() model.Outbound { return &openai.ImageGenerationOutbound{} },
	OutboundTypeAnthropic:             func() model.Outbound { return &authropic.MessageOutbound{} },
	OutboundTypeGemini:                func() model.Outbound { return &gemini.MessagesOutbound{} },
	OutboundTypeVolcengine:            func() model.Outbound { return &volcengine.ResponseOutbound{} },
}

func Get(outboundType OutboundType) model.Outbound {
	if factory, ok := outboundFactories[outboundType]; ok {
		return factory()
	}
	return nil
}

// GetForModel 获取出站适配器，对 Zen 渠道按模型名称动态路由到正确的协议适配器。
// Zen 路由规则：
//   - claude-* → Anthropic Messages 格式 (/zen/v1/messages)
//   - gpt-*    → OpenAI Responses 格式 (/zen/v1/responses)
//   - gemini-* → Gemini 格式
//   - 其他     → OpenAI Chat 格式 (/zen/v1/chat/completions)
func GetForModel(channelType OutboundType, modelName string) model.Outbound {
	if channelType != OutboundTypeZen {
		return Get(channelType)
	}
	lower := strings.ToLower(modelName)
	switch {
	case strings.HasPrefix(lower, "claude-"):
		return Get(OutboundTypeAnthropic)
	case strings.HasPrefix(lower, "gpt-"):
		return Get(OutboundTypeOpenAIResponse)
	case strings.HasPrefix(lower, "gemini-"):
		return Get(OutboundTypeGemini)
	default:
		return Get(OutboundTypeOpenAIChat)
	}
}
