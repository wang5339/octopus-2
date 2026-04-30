package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound/responsebody"
)

type ImageGenerationOutbound struct{}

// OpenAIImageGenerationUpstreamRequest 是发送给上游的 image generation 请求格式
type OpenAIImageGenerationUpstreamRequest struct {
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

// OpenAIImageGenerationUpstreamResponse 是上游返回的 image generation 响应格式
type OpenAIImageGenerationUpstreamResponse struct {
	Created      int64               `json:"created"`
	Data         []model.ImageObject `json:"data"`
	Background   *string             `json:"background,omitempty"`
	OutputFormat *string             `json:"output_format,omitempty"`
	Quality      *string             `json:"quality,omitempty"`
	Size         *string             `json:"size,omitempty"`
	Usage        *model.Usage        `json:"usage,omitempty"`
}

func (o *ImageGenerationOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseUrl, key string) (*http.Request, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	if request.ImageGenerationPrompt == nil {
		return nil, errors.New("not an image generation request")
	}

	// 构建 image generation 请求体
	imgReq := OpenAIImageGenerationUpstreamRequest{
		Model:             request.Model,
		Prompt:            *request.ImageGenerationPrompt,
		N:                 request.ImageGenerationN,
		Size:              request.ImageGenerationSize,
		Quality:           request.ImageGenerationQuality,
		ResponseFormat:    request.ImageGenerationResponseFormat,
		Style:             request.ImageGenerationStyle,
		OutputFormat:      request.ImageGenerationOutputFormat,
		Background:        request.ImageGenerationBackground,
		Moderation:        request.ImageGenerationModeration,
		OutputCompression: request.ImageGenerationOutputCompression,
		PartialImages:     request.ImageGenerationPartialImages,
		User:              request.User,
	}

	body, err := json.Marshal(imgReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	parsedUrl, err := url.Parse(strings.TrimSuffix(baseUrl, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}
	parsedUrl.Path = parsedUrl.Path + "/images/generations"
	req.URL = parsedUrl
	req.Method = http.MethodPost
	return req, nil
}

func (o *ImageGenerationOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
	body, err := responsebody.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	var openAIResp OpenAIImageGenerationUpstreamResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 转换为内部格式
	resp := &model.InternalLLMResponse{
		Created:                     openAIResp.Created,
		Object:                      "list",
		ImageData:                   openAIResp.Data,
		Usage:                       openAIResp.Usage,
		ImageGenerationBackground:   openAIResp.Background,
		ImageGenerationOutputFormat: openAIResp.OutputFormat,
		ImageGenerationQuality:      openAIResp.Quality,
		ImageGenerationSize:         openAIResp.Size,
	}

	return resp, nil
}

func (o *ImageGenerationOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
	// Image Generation API does not support streaming
	return nil, errors.New("streaming is not supported for image generation API")
}
