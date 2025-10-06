package anthropic

import (
	"testing"

	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// TestAppendErrorToRequest_Anthropic ensures Anthropic messages ([]MessageContent) work correctly
func TestAppendErrorToRequest_Anthropic(t *testing.T) {
	// Use the Anthropic provider's handler (which implements custom logic)
	provider := FromAnthropic(&anthropic.Client{})

	request := anthropic.MessagesRequest{
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewTextMessageContent("What is 2+2?"),
				},
			},
		},
	}

	// Provider returns custom result
	result := provider.AppendErrorToRequest(request, `{"answer": "five"}`, "JSON parsing failed: invalid number. Fix the syntax and retry.")
	if result == nil {
		t.Fatal("Anthropic provider should return a custom result, not nil")
	}

	resultReq, ok := result.(anthropic.MessagesRequest)
	if !ok {
		t.Fatalf("expected anthropic.MessagesRequest, got %T", result)
	}

	// Should have 3 messages: original user + assistant error + user correction
	if len(resultReq.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(resultReq.Messages))
	}

	// Check assistant message
	if resultReq.Messages[1].Role != anthropic.RoleAssistant {
		t.Errorf("message[1] role = %v, want assistant", resultReq.Messages[1].Role)
	}
	if len(resultReq.Messages[1].Content) != 1 {
		t.Fatalf("message[1] content length = %d, want 1", len(resultReq.Messages[1].Content))
	}
	if resultReq.Messages[1].Content[0].Type != anthropic.MessagesContentTypeText {
		t.Errorf("message[1] content[0] type = %v, want text", resultReq.Messages[1].Content[0].Type)
	}
	if resultReq.Messages[1].Content[0].Text == nil || *resultReq.Messages[1].Content[0].Text != `{"answer": "five"}` {
		t.Errorf("message[1] content[0] text = %v, want failed response", resultReq.Messages[1].Content[0].Text)
	}

	// Check user error message
	if resultReq.Messages[2].Role != anthropic.RoleUser {
		t.Errorf("message[2] role = %v, want user", resultReq.Messages[2].Role)
	}
	if len(resultReq.Messages[2].Content) != 1 {
		t.Fatalf("message[2] content length = %d, want 1", len(resultReq.Messages[2].Content))
	}
	if resultReq.Messages[2].Content[0].Type != anthropic.MessagesContentTypeText {
		t.Errorf("message[2] content[0] type = %v, want text", resultReq.Messages[2].Content[0].Type)
	}
	if resultReq.Messages[2].Content[0].Text == nil || *resultReq.Messages[2].Content[0].Text != "JSON parsing failed: invalid number. Fix the syntax and retry." {
		t.Errorf("message[2] content[0] text = %v, want error message", resultReq.Messages[2].Content[0].Text)
	}
}

// TestAppendErrorToRequest_Anthropic_Pointer ensures pointer requests work correctly
func TestAppendErrorToRequest_Anthropic_Pointer(t *testing.T) {
	provider := FromAnthropic(&anthropic.Client{})

	request := &anthropic.MessagesRequest{
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewTextMessageContent("What is 2+2?"),
				},
			},
		},
	}

	result := provider.AppendErrorToRequest(request, `{"answer": "five"}`, "JSON parsing failed: invalid number. Fix the syntax and retry.")
	if result == nil {
		t.Fatal("Anthropic provider should return a custom result, not nil")
	}

	resultReq, ok := result.(*anthropic.MessagesRequest)
	if !ok {
		t.Fatalf("expected *anthropic.MessagesRequest, got %T", result)
	}

	// Should have 3 messages
	if len(resultReq.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(resultReq.Messages))
	}
}
