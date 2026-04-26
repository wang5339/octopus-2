package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound/gemini"
	"github.com/google/uuid"
)

const (
	loadCodeAssistPath = "/v1internal:loadCodeAssist"
	generatePath       = "/v1internal:generateContent"
	streamPath         = "/v1internal:streamGenerateContent"
)

// codeAssistHeaders are required by the cloudcode-pa.googleapis.com API.
var codeAssistHeaders = map[string]string{
	"X-Goog-Api-Client": "gl-node/22.17.0",
	"Client-Metadata":   "ideType=IDE_UNSPECIFIED,platform=PLATFORM_UNSPECIFIED,pluginType=GEMINI",
}

// projectIDCache stores the managed project ID per OAuth token (first 64 chars as key).
var projectIDCache sync.Map

// antigravityWrappedRequest is the envelope format required by v1internal endpoints.
type antigravityWrappedRequest struct {
	Project      string                              `json:"project"`
	Model        string                              `json:"model"`
	UserPromptID string                              `json:"user_prompt_id"`
	Request      *model.GeminiGenerateContentRequest `json:"request"`
}

// antigravityWrappedResponse is the envelope returned by v1internal endpoints.
type antigravityWrappedResponse struct {
	Response *model.GeminiGenerateContentResponse `json:"response"`
	TraceID  string                               `json:"traceId,omitempty"`
}

// loadCodeAssistResponse is the relevant subset of the loadCodeAssist payload.
type loadCodeAssistResponse struct {
	CloudAiCompanionProject interface{} `json:"cloudaicompanionProject"`
}

// MessagesOutbound handles the Antigravity (Google Gemini Code Assist via OAuth) protocol.
// Key format: "<oauth_token>" or "<oauth_token>|<projectId>"
type MessagesOutbound struct{}

// parseKey splits the key into the OAuth token and optional project ID.
func parseKey(key string) (token, projectID string) {
	parts := strings.SplitN(key, "|", 2)
	token = parts[0]
	if len(parts) == 2 {
		projectID = parts[1]
	}
	return
}

// tokenCacheKey returns a cache key from the token (first 64 chars suffice for uniqueness).
func tokenCacheKey(token string) string {
	if len(token) > 64 {
		return token[:64]
	}
	return token
}

// getProjectID returns the managed project ID for the given token.
// It checks the in-memory cache first, then calls loadCodeAssist.
func getProjectID(ctx context.Context, baseURL, key string) (string, error) {
	token, projectID := parseKey(key)
	if projectID != "" {
		return projectID, nil
	}

	cacheKey := tokenCacheKey(token)
	if cached, ok := projectIDCache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	// Fetch from API
	reqBody := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(baseURL, "/")+loadCodeAssistPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create loadCodeAssist request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range codeAssistHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("loadCodeAssist request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("loadCodeAssist returned %d: %s", resp.StatusCode, string(body))
	}

	var payload loadCodeAssistResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode loadCodeAssist response: %w", err)
	}

	// Extract project ID from cloudaicompanionProject (string or object with "id")
	switch v := payload.CloudAiCompanionProject.(type) {
	case string:
		projectID = strings.TrimSpace(v)
	case map[string]interface{}:
		if id, ok := v["id"].(string); ok {
			projectID = strings.TrimSpace(id)
		}
	}

	if projectID != "" {
		projectIDCache.Store(cacheKey, projectID)
	}
	return projectID, nil
}

func (o *MessagesOutbound) TransformRequest(ctx context.Context, request *model.InternalLLMRequest, baseURL, key string) (*http.Request, error) {
	token, _ := parseKey(key)

	projectID, err := getProjectID(ctx, baseURL, key)
	if err != nil {
		return nil, fmt.Errorf("antigravity: failed to get project ID: %w", err)
	}

	// Build inner Gemini request (reuses gemini package conversion)
	geminiReq := gemini.ConvertLLMToGeminiRequest(request)

	// Determine action
	isStream := request.Stream != nil && *request.Stream
	pathSuffix := generatePath
	if isStream {
		pathSuffix = streamPath
	}

	// Build wrapped body
	modelName := request.Model
	// Strip "models/" prefix if present (API expects bare model name)
	modelName = strings.TrimPrefix(modelName, "models/")

	wrapped := &antigravityWrappedRequest{
		Project:      projectID,
		Model:        modelName,
		UserPromptID: uuid.New().String(),
		Request:      geminiReq,
	}

	body, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("antigravity: failed to marshal request: %w", err)
	}

	urlStr := strings.TrimSuffix(baseURL, "/") + pathSuffix
	if isStream {
		urlStr += "?alt=sse"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("antigravity: failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if isStream {
		req.Header.Set("Accept", "text/event-stream")
	}
	for k, v := range codeAssistHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (o *MessagesOutbound) TransformResponse(ctx context.Context, response *http.Response) (*model.InternalLLMResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("antigravity: failed to read response body: %w", err)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("antigravity: empty response body")
	}

	var wrapped antigravityWrappedResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("antigravity: failed to unmarshal response: %w", err)
	}
	if wrapped.Response == nil {
		return nil, fmt.Errorf("antigravity: missing inner response field")
	}

	return gemini.ConvertGeminiToLLMResponse(wrapped.Response, false), nil
}

func (o *MessagesOutbound) TransformStream(ctx context.Context, eventData []byte) (*model.InternalLLMResponse, error) {
	if bytes.HasPrefix(eventData, []byte("[DONE]")) || len(eventData) == 0 {
		return &model.InternalLLMResponse{Object: "[DONE]"}, nil
	}

	var wrapped antigravityWrappedResponse
	if err := json.Unmarshal(eventData, &wrapped); err != nil {
		return nil, fmt.Errorf("antigravity: failed to unmarshal stream chunk: %w", err)
	}
	if wrapped.Response == nil {
		return nil, fmt.Errorf("antigravity: missing inner response field in stream chunk")
	}

	return gemini.ConvertGeminiToLLMResponse(wrapped.Response, true), nil
}
