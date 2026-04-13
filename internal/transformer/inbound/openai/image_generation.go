package openai

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bestruirui/octopus/internal/transformer/model"
)

type ImageGenerationInbound struct {
	// storedResponse stores the non-stream response
	storedResponse *model.InternalLLMResponse
}

// OpenAIImageGenerationRequest 是 OpenAI 标准的 image generation 请求格式
type OpenAIImageGenerationRequest struct {
	Model             string  `json:"model"`
	Prompt            string  `json:"prompt"`
	N                 *int64  `json:"n,omitempty"`
	Size              *string `json:"size,omitempty"`
	Quality           *string `json:"quality,omitempty"`
	ResponseFormat    *string `json:"response_format,omitempty"`
	Style             *string `json:"style,omitempty"`
	User              *string `json:"user,omitempty"`
	OutputFormat      *string `json:"output_format,omitempty"`
	Background        *string `json:"background,omitempty"`
	Moderation        *string `json:"moderation,omitempty"`
	OutputCompression *int64  `json:"output_compression,omitempty"`
	PartialImages     *int64  `json:"partial_images,omitempty"`
}

// OpenAIImageGenerationResponse 是 OpenAI 标准的 image generation 响应格式
type OpenAIImageGenerationResponse struct {
	Created      int64               `json:"created"`
	Data         []model.ImageObject `json:"data"`
	Background   *string             `json:"background,omitempty"`
	OutputFormat *string             `json:"output_format,omitempty"`
	Quality      *string             `json:"quality,omitempty"`
	Size         *string             `json:"size,omitempty"`
	Usage        *model.Usage        `json:"usage,omitempty"`
}

func (i *ImageGenerationInbound) TransformRequest(ctx context.Context, body []byte) (*model.InternalLLMRequest, error) {
	var openAIReq OpenAIImageGenerationRequest
	if err := json.Unmarshal(body, &openAIReq); err != nil {
		return nil, err
	}

	if openAIReq.Prompt == "" {
		return nil, errors.New("prompt is required")
	}

	// 转换为内部格式
	var request model.InternalLLMRequest
	request.Model = openAIReq.Model
	request.ImageGenerationPrompt = &openAIReq.Prompt
	request.ImageGenerationN = openAIReq.N
	request.ImageGenerationSize = openAIReq.Size
	request.ImageGenerationQuality = openAIReq.Quality
	request.ImageGenerationResponseFormat = openAIReq.ResponseFormat
	request.ImageGenerationStyle = openAIReq.Style
	request.ImageGenerationOutputFormat = openAIReq.OutputFormat
	request.ImageGenerationBackground = openAIReq.Background
	request.ImageGenerationModeration = openAIReq.Moderation
	request.ImageGenerationOutputCompression = openAIReq.OutputCompression
	request.ImageGenerationPartialImages = openAIReq.PartialImages
	request.User = openAIReq.User
	request.RawAPIFormat = model.APIFormatOpenAIImageGeneration

	return &request, nil
}

func (i *ImageGenerationInbound) TransformResponse(ctx context.Context, response *model.InternalLLMResponse) ([]byte, error) {
	// Store the response for later retrieval
	i.storedResponse = response

	// 转换为 OpenAI 标准格式
	openAIResp := OpenAIImageGenerationResponse{
		Created:      response.Created,
		Data:         response.ImageData,
		Usage:        response.Usage,
		Background:   response.ImageGenerationBackground,
		OutputFormat: response.ImageGenerationOutputFormat,
		Quality:      response.ImageGenerationQuality,
		Size:         response.ImageGenerationSize,
	}

	if openAIResp.Created == 0 {
		openAIResp.Created = time.Now().Unix()
	}
	if openAIResp.Data == nil {
		openAIResp.Data = []model.ImageObject{}
	}

	body, err := json.Marshal(openAIResp)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (i *ImageGenerationInbound) TransformStream(ctx context.Context, stream *model.InternalLLMResponse) ([]byte, error) {
	// Image Generation API does not support streaming
	return nil, errors.New("streaming is not supported for image generation API")
}

// GetInternalResponse returns the complete internal response for logging, statistics, etc.
func (i *ImageGenerationInbound) GetInternalResponse(ctx context.Context) (*model.InternalLLMResponse, error) {
	return i.storedResponse, nil
}
