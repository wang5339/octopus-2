package outbound

import (
	"strings"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound/antigravity"
	"github.com/bestruirui/octopus/internal/transformer/outbound/authropic"
	"github.com/bestruirui/octopus/internal/transformer/outbound/copilot"
	"github.com/bestruirui/octopus/internal/transformer/outbound/gemini"
	"github.com/bestruirui/octopus/internal/transformer/outbound/openai"
	"github.com/bestruirui/octopus/internal/transformer/outbound/volcengine"
)

type OutboundType int

const (
	OutboundTypeOpenAIChat OutboundType = iota
	OutboundTypeOpenAIResponse
	OutboundTypeAnthropic
	OutboundTypeGemini
	OutboundTypeVolcengine
	OutboundTypeOpenAIEmbedding
	OutboundTypeOpenAIImageGeneration
	OutboundTypeGithubCopilot // 7: GitHub Copilot (OAuth Device Flow, uses OpenAI Chat format)
	OutboundTypeAntigravity   // 8: Antigravity (OAuth Web Flow reverse proxy)
	OutboundTypeZen           // 9: OpenCode Zen (model-aware protocol routing: Claude→Anthropic, GPT→Responses, Gemini→Gemini, others→Chat)
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
	OutboundTypeGithubCopilot:  true,
	OutboundTypeAntigravity:    true,
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
	// GitHub Copilot exchanges OAuth token for short-lived Copilot API token, then uses OpenAI Chat format
	OutboundTypeGithubCopilot: func() model.Outbound { return &copilot.ChatOutbound{} },
	// Antigravity uses Google Gemini Code Assist API via OAuth Bearer token
	OutboundTypeAntigravity: func() model.Outbound { return &antigravity.MessagesOutbound{} },
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
