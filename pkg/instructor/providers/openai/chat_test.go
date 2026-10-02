package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	openaiSDK "github.com/sashabaranov/go-openai"
)

type testResponse struct {
	OK bool `json:"ok"`
}

type emptyChoicesHTTPClient struct {
	t *testing.T
}

func (c emptyChoicesHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.t.Helper()
	if req.URL.Path != "/v1/chat/completions" {
		c.t.Fatalf("unexpected request path: %s", req.URL.Path)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"id": "chatcmpl-empty",
			"object": "chat.completion",
			"created": 1,
			"model": "test-model",
			"choices": [],
			"usage": {
				"prompt_tokens": 3,
				"completion_tokens": 0,
				"total_tokens": 3
			}
		}`)),
	}, nil
}

func TestFirstChoiceMessageContent(t *testing.T) {
	t.Run("empty choices returns error", func(t *testing.T) {
		_, err := firstChoiceMessageContent(&openaiSDK.ChatCompletionResponse{})
		if err == nil {
			t.Fatal("expected error for empty choices")
		}
	})

	t.Run("returns first choice content", func(t *testing.T) {
		content, err := firstChoiceMessageContent(&openaiSDK.ChatCompletionResponse{
			Choices: []openaiSDK.ChatCompletionChoice{
				{
					Message: openaiSDK.ChatCompletionMessage{
						Content: `{"ok":true}`,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != `{"ok":true}` {
			t.Fatalf("content = %q, want %q", content, `{"ok":true}`)
		}
	})
}

func TestCreateChatCompletionJSONSchemaReturnsErrorForEmptyChoices(t *testing.T) {
	config := openaiSDK.DefaultConfig("test-key")
	config.BaseURL = "http://example.test/v1"
	config.HTTPClient = emptyChoicesHTTPClient{t: t}
	client := FromOpenAI(
		openaiSDK.NewClientWithConfig(config),
		core.WithMode(core.ModeJSONSchema),
		core.WithMaxRetries(0),
	)

	var result testResponse
	resp, err := client.CreateChatCompletion(context.Background(), openaiSDK.ChatCompletionRequest{
		Model: "test-model",
		Messages: []openaiSDK.ChatCompletionMessage{
			{
				Role:    openaiSDK.ChatMessageRoleUser,
				Content: "return json",
			},
		},
	}, &result)

	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "received no choices from model") {
		t.Fatalf("error = %q, want empty choices error", err.Error())
	}
	if resp.Usage.PromptTokens != 3 || resp.Usage.TotalTokens != 3 {
		t.Fatalf("usage = %+v, want prompt and total tokens preserved", resp.Usage)
	}
	if result.OK {
		t.Fatal("expected result to remain zero value")
	}
}
