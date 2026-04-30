package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/dlclark/regexp2"
)

const (
	fetchModelsMaxResponseBytes = 4 << 20 // 4 MiB，避免异常上游返回超大响应拖垮进程。
	fetchModelsMaxErrorBytes    = 1024
)

func FetchModels(ctx context.Context, request model.Channel) ([]string, error) {
	client, err := ChannelHttpClient(&request)
	if err != nil {
		return nil, err
	}
	fetchModel := make([]string, 0)
	switch request.Type {
	case outbound.OutboundTypeAnthropic:
		fetchModel, err = fetchAnthropicModels(client, ctx, request)
	case outbound.OutboundTypeGemini:
		fetchModel, err = fetchGeminiModels(client, ctx, request)
	default:
		fetchModel, err = fetchOpenAIModels(client, ctx, request)
	}
	if err != nil {
		return nil, err
	}
	if request.MatchRegex != nil && *request.MatchRegex != "" {
		matchModel := make([]string, 0)
		re, err := regexp2.Compile(*request.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			return nil, err
		}
		for _, model := range fetchModel {
			matched, err := re.MatchString(model)
			if err != nil {
				return nil, err
			}
			if matched {
				matchModel = append(matchModel, model)
			}
		}
		return matchModel, nil
	}
	return fetchModel, nil
}

// refer: https://platform.openai.com/docs/api-reference/models/list
func fetchOpenAIModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fetchModelsURL(request),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create OpenAI models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+request.GetChannelKey().ChannelKey)
	applyFetchModelsCustomHeaders(req, request)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	var result model.OpenAIModelList

	if err := decodeFetchModelsResponse(resp, "OpenAI", &result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

// refer: https://ai.google.dev/api/models
func fetchGeminiModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	var allModels []string
	pageToken := ""
	maxPages := 100 // 防止无限循环
	pageCount := 0

	for {
		if pageCount >= maxPages {
			return nil, fmt.Errorf("exceeded maximum page limit (%d) when fetching Gemini models", maxPages)
		}
		pageCount++

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fetchModelsURL(request),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create Gemini models request: %w", err)
		}
		req.Header.Set("X-Goog-Api-Key", request.GetChannelKey().ChannelKey)
		applyFetchModelsCustomHeaders(req, request)
		if pageToken != "" {
			q := req.URL.Query()
			q.Add("pageToken", pageToken)
			req.URL.RawQuery = q.Encode()
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var result model.GeminiModelList

		if err := decodeFetchModelsResponse(resp, "Gemini", &result); err != nil {
			return nil, err
		}

		for _, m := range result.Models {
			name := strings.TrimPrefix(m.Name, "models/")
			allModels = append(allModels, name)
		}

		nextPageToken := strings.TrimSpace(result.NextPageToken)
		if nextPageToken == "" {
			break
		}
		if nextPageToken == pageToken {
			return nil, fmt.Errorf("Gemini models pagination did not advance: pageToken %q", pageToken)
		}
		pageToken = nextPageToken
	}
	if len(allModels) == 0 {
		return fetchOpenAIModels(client, ctx, request)
	}
	return allModels, nil
}

// refer: https://platform.claude.com/docs
func fetchAnthropicModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {

	var allModels []string
	var afterID string
	maxPages := 100 // 防止无限循环
	pageCount := 0

	for {
		if pageCount >= maxPages {
			return nil, fmt.Errorf("exceeded maximum page limit (%d) when fetching Anthropic models", maxPages)
		}
		pageCount++

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fetchModelsURL(request),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create Anthropic models request: %w", err)
		}
		req.Header.Set("X-Api-Key", request.GetChannelKey().ChannelKey)
		req.Header.Set("Anthropic-Version", "2023-06-01")
		applyFetchModelsCustomHeaders(req, request)
		// 设置多页参数
		q := req.URL.Query()

		if afterID != "" {
			q.Set("after_id", afterID)
		}
		req.URL.RawQuery = q.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var result model.AnthropicModelList

		if err := decodeFetchModelsResponse(resp, "Anthropic", &result); err != nil {
			return nil, err
		}

		for _, m := range result.Data {
			allModels = append(allModels, m.ID)
		}

		if !result.HasMore {
			break
		}

		nextAfterID := strings.TrimSpace(result.LastID)
		if nextAfterID == "" {
			return nil, fmt.Errorf("Anthropic models pagination has_more=true but last_id is empty")
		}
		if nextAfterID == afterID {
			return nil, fmt.Errorf("Anthropic models pagination did not advance: after_id %q", afterID)
		}
		afterID = nextAfterID
	}
	if len(allModels) == 0 {
		return fetchOpenAIModels(client, ctx, request)
	}
	return allModels, nil
}

func fetchModelsURL(request model.Channel) string {
	return strings.TrimRight(request.GetBaseUrl(), "/") + "/models"
}

func applyFetchModelsCustomHeaders(req *http.Request, request model.Channel) {
	for _, header := range request.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}
}

func decodeFetchModelsResponse(resp *http.Response, provider string, dst any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchModelsMaxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read %s models response: %w", provider, err)
	}
	if len(body) > fetchModelsMaxResponseBytes {
		return fmt.Errorf("%s models response exceeds %d bytes", provider, fetchModelsMaxResponseBytes)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s models request failed: %s: %s", provider, resp.Status, fetchModelsErrorBody(body))
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode %s models response: %w", provider, err)
	}
	return nil
}

func fetchModelsErrorBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "<empty body>"
	}
	if len(text) > fetchModelsMaxErrorBytes {
		return text[:fetchModelsMaxErrorBytes] + "...<truncated>"
	}
	return text
}
