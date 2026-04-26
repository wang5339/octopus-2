package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
	"github.com/bestruirui/octopus/internal/transformer/outbound/copilot"
	"github.com/dlclark/regexp2"
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
	case outbound.OutboundTypeAntigravity:
		fetchModel, err = fetchAntigravityModels(client, ctx, request)
	case outbound.OutboundTypeGithubCopilot:
		fetchModel, err = fetchCopilotModels(client, ctx, request)
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
	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		request.GetBaseUrl()+"/models",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+request.GetChannelKey().ChannelKey)
	for _, header := range request.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.OpenAIModelList

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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

		req, _ := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			request.GetBaseUrl()+"/models",
			nil,
		)
		req.Header.Set("X-Goog-Api-Key", request.GetChannelKey().ChannelKey)
		for _, header := range request.CustomHeader {
			if header.HeaderKey != "" {
				req.Header.Set(header.HeaderKey, header.HeaderValue)
			}
		}
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

		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, m := range result.Models {
			name := strings.TrimPrefix(m.Name, "models/")
			allModels = append(allModels, name)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
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

		req, _ := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			request.GetBaseUrl()+"/models",
			nil,
		)
		req.Header.Set("X-Api-Key", request.GetChannelKey().ChannelKey)
		req.Header.Set("Anthropic-Version", "2023-06-01")
		for _, header := range request.CustomHeader {
			if header.HeaderKey != "" {
				req.Header.Set(header.HeaderKey, header.HeaderValue)
			}
		}
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

		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, m := range result.Data {
			allModels = append(allModels, m.ID)
		}

		if !result.HasMore {
			break
		}

		afterID = result.LastID
	}
	if len(allModels) == 0 {
		return fetchOpenAIModels(client, ctx, request)
	}
	return allModels, nil
}

// fetchAntigravityModels retrieves models for Antigravity (Google Gemini Code Assist via OAuth).
// It calls POST /v1internal:retrieveUserQuota to get quota buckets, each containing a modelId.
// Key format: "<oauth_token>" or "<oauth_token>|<projectId>"
func fetchAntigravityModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	key := request.GetChannelKey().ChannelKey
	keyParts := strings.SplitN(key, "|", 2)
	token := keyParts[0]

	// Determine project ID: from key suffix or from loadCodeAssist
	projectID := ""
	if len(keyParts) == 2 {
		projectID = keyParts[1]
	} else {
		// Call loadCodeAssist to get the managed project ID
		loadBody := `{"metadata":{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}}`
		loadReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
			request.GetBaseUrl()+"/v1internal:loadCodeAssist",
			strings.NewReader(loadBody))
		if err != nil {
			return nil, err
		}
		loadReq.Header.Set("Authorization", "Bearer "+token)
		loadReq.Header.Set("Content-Type", "application/json")
		loadReq.Header.Set("X-Goog-Api-Client", "gl-node/22.17.0")
		loadReq.Header.Set("Client-Metadata", "ideType=IDE_UNSPECIFIED,platform=PLATFORM_UNSPECIFIED,pluginType=GEMINI")

		loadResp, err := client.Do(loadReq)
		if err != nil {
			return nil, err
		}
		defer loadResp.Body.Close()

		var loadPayload struct {
			CloudAiCompanionProject interface{} `json:"cloudaicompanionProject"`
		}
		if err := json.NewDecoder(loadResp.Body).Decode(&loadPayload); err != nil {
			return nil, err
		}
		switch v := loadPayload.CloudAiCompanionProject.(type) {
		case string:
			projectID = strings.TrimSpace(v)
		case map[string]interface{}:
			if id, ok := v["id"].(string); ok {
				projectID = strings.TrimSpace(id)
			}
		}
	}

	// Call retrieveUserQuota to get available models
	quotaBody := `{"project":"` + projectID + `"}`
	quotaReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		request.GetBaseUrl()+"/v1internal:retrieveUserQuota",
		strings.NewReader(quotaBody))
	if err != nil {
		return nil, err
	}
	quotaReq.Header.Set("Authorization", "Bearer "+token)
	quotaReq.Header.Set("Content-Type", "application/json")
	quotaReq.Header.Set("X-Goog-Api-Client", "gl-node/22.17.0")
	quotaReq.Header.Set("Client-Metadata", "ideType=IDE_UNSPECIFIED,platform=PLATFORM_UNSPECIFIED,pluginType=GEMINI")
	for _, header := range request.CustomHeader {
		if header.HeaderKey != "" {
			quotaReq.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	quotaResp, err := client.Do(quotaReq)
	if err != nil {
		return nil, err
	}
	defer quotaResp.Body.Close()

	var quotaPayload struct {
		Buckets []struct {
			ModelID string `json:"modelId"`
		} `json:"buckets"`
	}
	if err := json.NewDecoder(quotaResp.Body).Decode(&quotaPayload); err != nil {
		return nil, err
	}

	// Deduplicate model IDs
	seen := make(map[string]bool)
	var models []string
	for _, bucket := range quotaPayload.Buckets {
		if bucket.ModelID != "" && !seen[bucket.ModelID] {
			seen[bucket.ModelID] = true
			models = append(models, bucket.ModelID)
		}
	}
	return models, nil
}

// fetchCopilotModels retrieves models for GitHub Copilot channel.
// It first exchanges the GitHub OAuth token for a short-lived Copilot API token,
// then calls the OpenAI-compatible /models endpoint on api.githubcopilot.com.
func fetchCopilotModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
	githubToken := request.GetChannelKey().ChannelKey
	copilotToken, err := copilot.ExchangeToken(ctx, githubToken)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		request.GetBaseUrl()+"/models",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+copilotToken)
	for _, header := range request.CustomHeader {
		if header.HeaderKey != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.OpenAIModelList

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}
