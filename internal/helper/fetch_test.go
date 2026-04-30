package helper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/transformer/outbound"
)

func TestFetchOpenAIModelsRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer test-key"; got != want {
			t.Fatalf("Authorization header = %q, want %q", got, want)
		}
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := fetchOpenAIModels(server.Client(), context.Background(), fetchTestChannel(server.URL, outbound.OutboundTypeOpenAIChat))
	if err == nil {
		t.Fatal("fetchOpenAIModels returned nil error for non-2xx response")
	}
	assertErrorContains(t, err, "OpenAI models request failed: 401 Unauthorized")
	assertErrorContains(t, err, "bad key")
}

func TestFetchOpenAIModelsRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", fetchModelsMaxResponseBytes+1)))
	}))
	defer server.Close()

	_, err := fetchOpenAIModels(server.Client(), context.Background(), fetchTestChannel(server.URL, outbound.OutboundTypeOpenAIChat))
	if err == nil {
		t.Fatal("fetchOpenAIModels returned nil error for oversized response")
	}
	assertErrorContains(t, err, fmt.Sprintf("OpenAI models response exceeds %d bytes", fetchModelsMaxResponseBytes))
}

func TestFetchGeminiModelsDetectsStuckPageToken(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("pageToken") {
		case "":
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-1.5-pro"}],"nextPageToken":"same"}`))
		case "same":
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-1.5-flash"}],"nextPageToken":"same"}`))
		default:
			t.Fatalf("unexpected pageToken %q", r.URL.Query().Get("pageToken"))
		}
	}))
	defer server.Close()

	_, err := fetchGeminiModels(server.Client(), context.Background(), fetchTestChannel(server.URL, outbound.OutboundTypeGemini))
	if err == nil {
		t.Fatal("fetchGeminiModels returned nil error for stuck pagination token")
	}
	assertErrorContains(t, err, `Gemini models pagination did not advance: pageToken "same"`)
	if calls != 2 {
		t.Fatalf("server calls = %d, want 2", calls)
	}
}

func TestFetchAnthropicModelsRejectsHasMoreWithoutLastID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Anthropic-Version"), "2023-06-01"; got != want {
			t.Fatalf("Anthropic-Version header = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-3-5-sonnet","type":"model"}],"has_more":true,"last_id":""}`))
	}))
	defer server.Close()

	_, err := fetchAnthropicModels(server.Client(), context.Background(), fetchTestChannel(server.URL, outbound.OutboundTypeAnthropic))
	if err == nil {
		t.Fatal("fetchAnthropicModels returned nil error for has_more without last_id")
	}
	assertErrorContains(t, err, "Anthropic models pagination has_more=true but last_id is empty")
}

func TestFetchModelsRequestCreationError(t *testing.T) {
	channel := fetchTestChannel("://bad", outbound.OutboundTypeOpenAIChat)

	_, err := fetchOpenAIModels(http.DefaultClient, context.Background(), channel)
	if err == nil {
		t.Fatal("fetchOpenAIModels returned nil error for invalid base URL")
	}
	assertErrorContains(t, err, "create OpenAI models request")
}

func fetchTestChannel(baseURL string, channelType outbound.OutboundType) model.Channel {
	return model.Channel{
		Type: channelType,
		BaseUrls: []model.BaseUrl{
			{URL: baseURL},
		},
		Keys: []model.ChannelKey{
			{Enabled: true, ChannelKey: "test-key"},
		},
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error is nil, want substring %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}
