package core

import (
	"testing"

	openaiSDK "github.com/sashabaranov/go-openai"
)

// TestDefaultAppendErrorToRequest_OpenAI ensures OpenAI messages (string content) work correctly
func TestDefaultAppendErrorToRequest_OpenAI(t *testing.T) {
	request := openaiSDK.ChatCompletionRequest{
		Messages: []openaiSDK.ChatCompletionMessage{
			{
				Role:    openaiSDK.ChatMessageRoleUser,
				Content: "What is 2+2?",
			},
		},
	}

	result := defaultAppendErrorToRequest(request, `{"answer": "five"}`, "JSON parsing failed: invalid number. Fix the syntax and retry.")

	resultReq, ok := result.(openaiSDK.ChatCompletionRequest)
	if !ok {
		t.Fatalf("expected openai.ChatCompletionRequest, got %T", result)
	}

	// Should have 3 messages: original user + assistant error + user correction
	if len(resultReq.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(resultReq.Messages))
	}

	// Check assistant message
	if resultReq.Messages[1].Role != openaiSDK.ChatMessageRoleAssistant {
		t.Errorf("message[1] role = %v, want assistant", resultReq.Messages[1].Role)
	}
	if resultReq.Messages[1].Content != `{"answer": "five"}` {
		t.Errorf("message[1] content = %v, want failed response", resultReq.Messages[1].Content)
	}

	// Check user error message
	if resultReq.Messages[2].Role != openaiSDK.ChatMessageRoleUser {
		t.Errorf("message[2] role = %v, want user", resultReq.Messages[2].Role)
	}
	if resultReq.Messages[2].Content != "JSON parsing failed: invalid number. Fix the syntax and retry." {
		t.Errorf("message[2] content = %v, want error message", resultReq.Messages[2].Content)
	}
}

// TestDefaultAppendErrorToRequest_NoMessagesField ensures structs without Messages field are unchanged
func TestDefaultAppendErrorToRequest_NoMessagesField(t *testing.T) {
	// Test with a generic struct that doesn't have a Messages field
	type CustomRequest struct {
		Query   string
		Options map[string]any
	}

	request := CustomRequest{
		Query:   "What is 2+2?",
		Options: map[string]any{"temp": 0.7},
	}

	// The function should return the original request unchanged
	result := defaultAppendErrorToRequest(request, `{"answer": "five"}`, "JSON parsing failed: invalid number. Fix the syntax and retry.")

	resultReq, ok := result.(CustomRequest)
	if !ok {
		t.Fatalf("expected CustomRequest, got %T", result)
	}

	// Should be unchanged since it doesn't have "Messages" field
	if resultReq.Query != request.Query {
		t.Errorf("expected request to be unchanged")
	}
}
