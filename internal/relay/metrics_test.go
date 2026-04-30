package relay

import (
	"encoding/json"
	"strings"
	"testing"

	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
)

func TestFilterRequestForLogOmitsMediaPayloads(t *testing.T) {
	detail := "high"
	text := "describe this file"
	req := &transformerModel.InternalLLMRequest{
		Model: "gpt-4o-mini",
		Messages: []transformerModel.Message{
			{
				Role: "user",
				Content: transformerModel.MessageContent{
					MultipleContent: []transformerModel.MessageContentPart{
						{Type: "text", Text: &text},
						{
							Type: "image_url",
							ImageURL: &transformerModel.ImageURL{
								URL:    "data:image/png;base64,SECRET_REQUEST_IMAGE",
								Detail: &detail,
							},
						},
						{
							Type:  "input_audio",
							Audio: &transformerModel.Audio{Format: "wav", Data: "SECRET_REQUEST_AUDIO"},
						},
						{
							Type: "file",
							File: &transformerModel.File{Filename: "report.pdf", FileData: "SECRET_REQUEST_FILE"},
						},
					},
				},
				Images: []transformerModel.MessageContentPart{
					{
						Type:     "image_url",
						ImageURL: &transformerModel.ImageURL{URL: "data:image/png;base64,SECRET_REQUEST_IMAGES_SLICE"},
					},
				},
				Audio: relayMessageAudio("SECRET_REQUEST_MESSAGE_AUDIO"),
			},
		},
	}

	filtered := (&RelayMetrics{}).filterRequestForLog(req)
	body := mustMarshalRelayLogPayload(t, filtered)

	for _, secret := range []string{
		"SECRET_REQUEST_IMAGE",
		"SECRET_REQUEST_AUDIO",
		"SECRET_REQUEST_FILE",
		"SECRET_REQUEST_IMAGES_SLICE",
		"SECRET_REQUEST_MESSAGE_AUDIO",
	} {
		if strings.Contains(body, secret) {
			t.Fatalf("filtered request log still contains %q: %s", secret, body)
		}
	}
	for _, want := range []string{logOmittedImageData, logOmittedAudioData, logOmittedFileData, "report.pdf", "high"} {
		if !strings.Contains(body, want) {
			t.Fatalf("filtered request log does not contain %q: %s", want, body)
		}
	}
	if len(filtered.Messages[0].Images) != 0 {
		t.Fatalf("filtered request should drop Message.Images, got %d items", len(filtered.Messages[0].Images))
	}

	// 过滤只能影响日志副本，不能污染后续转发/统计仍在使用的原始请求对象。
	if got := req.Messages[0].Content.MultipleContent[1].ImageURL.URL; got != "data:image/png;base64,SECRET_REQUEST_IMAGE" {
		t.Fatalf("original request image_url was mutated: %q", got)
	}
	if got := req.Messages[0].Content.MultipleContent[2].Audio.Data; got != "SECRET_REQUEST_AUDIO" {
		t.Fatalf("original request input_audio was mutated: %q", got)
	}
	if got := req.Messages[0].Content.MultipleContent[3].File.FileData; got != "SECRET_REQUEST_FILE" {
		t.Fatalf("original request file_data was mutated: %q", got)
	}
	if got := req.Messages[0].Audio.Data; got != "SECRET_REQUEST_MESSAGE_AUDIO" {
		t.Fatalf("original request message audio was mutated: %q", got)
	}
	if len(req.Messages[0].Images) != 1 {
		t.Fatalf("original request images were mutated: got %d items", len(req.Messages[0].Images))
	}
}

func TestFilterResponseForLogOmitsMediaPayloads(t *testing.T) {
	url := "https://example.test/generated.png"
	b64JSON := "SECRET_RESPONSE_IMAGE_B64"
	embeddingBase64 := "SECRET_RESPONSE_EMBEDDING_B64"
	detail := "auto"
	resp := &transformerModel.InternalLLMResponse{
		ID:     "chatcmpl-test",
		Object: "chat.completion",
		Choices: []transformerModel.Choice{
			{
				Message: &transformerModel.Message{
					Role: "assistant",
					Content: transformerModel.MessageContent{
						MultipleContent: []transformerModel.MessageContentPart{
							{
								Type: "image_url",
								ImageURL: &transformerModel.ImageURL{
									URL:    "data:image/png;base64,SECRET_RESPONSE_MESSAGE_IMAGE",
									Detail: &detail,
								},
							},
							{
								Type:  "input_audio",
								Audio: &transformerModel.Audio{Format: "mp3", Data: "SECRET_RESPONSE_AUDIO"},
							},
							{
								Type: "file",
								File: &transformerModel.File{Filename: "answer.pdf", FileData: "SECRET_RESPONSE_FILE"},
							},
						},
					},
					Images: []transformerModel.MessageContentPart{
						{
							Type:     "image_url",
							ImageURL: &transformerModel.ImageURL{URL: "data:image/png;base64,SECRET_RESPONSE_IMAGES_SLICE"},
						},
					},
					Audio: relayMessageAudio("SECRET_RESPONSE_MESSAGE_AUDIO"),
				},
				Delta: &transformerModel.Message{
					Role: "assistant",
					Content: transformerModel.MessageContent{
						MultipleContent: []transformerModel.MessageContentPart{
							{
								Type:  "input_audio",
								Audio: &transformerModel.Audio{Format: "wav", Data: "SECRET_RESPONSE_DELTA_AUDIO"},
							},
						},
					},
				},
			},
		},
		ImageData: []transformerModel.ImageObject{
			{URL: &url, B64JSON: &b64JSON, RevisedPrompt: "keep this prompt"},
		},
		EmbeddingData: []transformerModel.EmbeddingObject{
			{
				Object: "embedding",
				Index:  0,
				Embedding: transformerModel.Embedding{
					FloatArray: []float64{0.1, 0.2, 0.3},
				},
			},
			{
				Object: "embedding",
				Index:  1,
				Embedding: transformerModel.Embedding{
					Base64String: &embeddingBase64,
				},
			},
		},
	}

	filtered := (&RelayMetrics{}).filterResponseForLog(resp)
	body := mustMarshalRelayLogPayload(t, filtered)

	for _, secret := range []string{
		"SECRET_RESPONSE_IMAGE_B64",
		"SECRET_RESPONSE_MESSAGE_IMAGE",
		"SECRET_RESPONSE_AUDIO",
		"SECRET_RESPONSE_FILE",
		"SECRET_RESPONSE_IMAGES_SLICE",
		"SECRET_RESPONSE_MESSAGE_AUDIO",
		"SECRET_RESPONSE_DELTA_AUDIO",
		"SECRET_RESPONSE_EMBEDDING_B64",
	} {
		if strings.Contains(body, secret) {
			t.Fatalf("filtered response log still contains %q: %s", secret, body)
		}
	}
	for _, want := range []string{
		logOmittedImageData,
		logOmittedAudioData,
		logOmittedFileData,
		logOmittedEmbeddingData,
		"answer.pdf",
		"keep this prompt",
		"https://example.test/generated.png",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("filtered response log does not contain %q: %s", want, body)
		}
	}

	// 过滤只能影响日志副本，不能污染已聚合的原始响应对象。
	if got := *resp.ImageData[0].B64JSON; got != "SECRET_RESPONSE_IMAGE_B64" {
		t.Fatalf("original response b64_json was mutated: %q", got)
	}
	if got := resp.EmbeddingData[0].Embedding.FloatArray; len(got) != 3 || got[0] != 0.1 {
		t.Fatalf("original response float embedding was mutated: %#v", got)
	}
	if got := *resp.EmbeddingData[1].Embedding.Base64String; got != "SECRET_RESPONSE_EMBEDDING_B64" {
		t.Fatalf("original response base64 embedding was mutated: %q", got)
	}
	if got := resp.Choices[0].Message.Content.MultipleContent[0].ImageURL.URL; got != "data:image/png;base64,SECRET_RESPONSE_MESSAGE_IMAGE" {
		t.Fatalf("original response image_url was mutated: %q", got)
	}
	if got := resp.Choices[0].Message.Content.MultipleContent[1].Audio.Data; got != "SECRET_RESPONSE_AUDIO" {
		t.Fatalf("original response input_audio was mutated: %q", got)
	}
	if got := resp.Choices[0].Message.Content.MultipleContent[2].File.FileData; got != "SECRET_RESPONSE_FILE" {
		t.Fatalf("original response file_data was mutated: %q", got)
	}
	if got := resp.Choices[0].Message.Audio.Data; got != "SECRET_RESPONSE_MESSAGE_AUDIO" {
		t.Fatalf("original response message audio was mutated: %q", got)
	}
	if got := resp.Choices[0].Delta.Content.MultipleContent[0].Audio.Data; got != "SECRET_RESPONSE_DELTA_AUDIO" {
		t.Fatalf("original response delta audio was mutated: %q", got)
	}
	if len(resp.Choices[0].Message.Images) != 1 {
		t.Fatalf("original response images were mutated: got %d items", len(resp.Choices[0].Message.Images))
	}
}

func relayMessageAudio(data string) *struct {
	Data       string `json:"data,omitempty"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
	ID         string `json:"id,omitempty"`
	Transcript string `json:"transcript,omitempty"`
} {
	return &struct {
		Data       string `json:"data,omitempty"`
		ExpiresAt  int64  `json:"expires_at,omitempty"`
		ID         string `json:"id,omitempty"`
		Transcript string `json:"transcript,omitempty"`
	}{
		Data:       data,
		ID:         "audio-id",
		Transcript: "kept transcript",
	}
}

func mustMarshalRelayLogPayload(t *testing.T, v any) string {
	t.Helper()

	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return string(body)
}
