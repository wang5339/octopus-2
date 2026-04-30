package helper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
)

var allowedParamOverrideKeys = map[string]struct{}{
	"audio":                               {},
	"embedding_dimensions":                {},
	"embedding_encoding_format":           {},
	"enable_thinking":                     {},
	"extra_body":                          {},
	"frequency_penalty":                   {},
	"image_generation_background":         {},
	"image_generation_moderation":         {},
	"image_generation_n":                  {},
	"image_generation_output_compression": {},
	"image_generation_output_format":      {},
	"image_generation_partial_images":     {},
	"image_generation_quality":            {},
	"image_generation_response_format":    {},
	"image_generation_size":               {},
	"image_generation_style":              {},
	"logit_bias":                          {},
	"logprobs":                            {},
	"max_completion_tokens":               {},
	"max_tokens":                          {},
	"metadata":                            {},
	"modalities":                          {},
	"parallel_tool_calls":                 {},
	"presence_penalty":                    {},
	"prompt_cache_key":                    {},
	"prompt_cache_retention":              {},
	"reasoning_effort":                    {},
	"response_format":                     {},
	"safety_identifier":                   {},
	"seed":                                {},
	"service_tier":                        {},
	"stop":                                {},
	"store":                               {},
	"temperature":                         {},
	"tool_choice":                         {},
	"top_logprobs":                        {},
	"top_p":                               {},
	"user":                                {},
}

// CloneInternalLLMRequest 深拷贝请求主体，同时保留 json:"-" 的辅助字段。
// 出站适配器会修改请求对象；每次转发前复制一份可以避免重试或不同渠道互相污染。
func CloneInternalLLMRequest(req *transformerModel.InternalLLMRequest) (*transformerModel.InternalLLMRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var cloned transformerModel.InternalLLMRequest
	if err := json.Unmarshal(body, &cloned); err != nil {
		return nil, err
	}
	copyInternalLLMRequestHelpFields(&cloned, req)
	return &cloned, nil
}

// ApplyParamOverride 将渠道级参数覆盖应用到 InternalLLMRequest。
// 只允许覆盖采样、输出、图片/embedding 等请求参数，不允许覆盖 model/messages/input/stream，
// 防止管理侧配置绕过路由、安全和响应处理边界。
func ApplyParamOverride(req *transformerModel.InternalLLMRequest, raw *string) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}

	var overrides map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(*raw)), &overrides); err != nil {
		return fmt.Errorf("invalid param_override json: %w", err)
	}
	for key := range overrides {
		if _, ok := allowedParamOverrideKeys[key]; !ok {
			return fmt.Errorf("param_override field %q is not allowed", key)
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(body, &merged); err != nil {
		return err
	}
	for key, value := range overrides {
		if string(value) == "null" {
			delete(merged, key)
			continue
		}
		merged[key] = value
	}

	mergedBody, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	var next transformerModel.InternalLLMRequest
	if err := json.Unmarshal(mergedBody, &next); err != nil {
		return fmt.Errorf("invalid param_override value: %w", err)
	}
	copyInternalLLMRequestHelpFields(&next, req)
	if err := next.Validate(); err != nil {
		return fmt.Errorf("param_override makes request invalid: %w", err)
	}
	*req = next
	return nil
}

func copyInternalLLMRequestHelpFields(dst, src *transformerModel.InternalLLMRequest) {
	dst.ReasoningBudget = src.ReasoningBudget
	dst.AdaptiveThinking = src.AdaptiveThinking
	dst.RawRequest = append([]byte(nil), src.RawRequest...)
	dst.RawAPIFormat = src.RawAPIFormat
	dst.TransformerMetadata = cloneStringMap(src.TransformerMetadata)
	dst.TransformOptions = cloneTransformOptions(src.TransformOptions)
	dst.Include = append([]string(nil), src.Include...)
	dst.PreviousResponseID = src.PreviousResponseID
	dst.Truncation = src.Truncation
	dst.Query = cloneURLValues(src.Query)
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneURLValues(src url.Values) url.Values {
	if src == nil {
		return nil
	}
	dst := make(url.Values, len(src))
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
	return dst
}

func cloneTransformOptions(src transformerModel.TransformOptions) transformerModel.TransformOptions {
	dst := src
	if src.ArrayInputs != nil {
		value := *src.ArrayInputs
		dst.ArrayInputs = &value
	}
	return dst
}
