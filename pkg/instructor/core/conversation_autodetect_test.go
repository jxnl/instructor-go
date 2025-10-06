package core

import (
	"encoding/json"
	"testing"

	anthropic "github.com/liushuangls/go-anthropic/v2"
	openai "github.com/sashabaranov/go-openai"
	genai "google.golang.org/genai"
)

func TestAddResponse_AutoDetectAnthropic(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create Anthropic response
	text := "I'll check the weather."
	inputData, _ := json.Marshal(map[string]any{"location": "SF"})
	resp := anthropic.MessagesResponse{
		Content: []anthropic.MessageContent{
			anthropic.NewTextMessageContent(text),
			anthropic.NewToolUseMessageContent("toolu_123", "get_weather", inputData),
		},
	}

	// Should auto-detect and add without setting handler
	err := conv.AddResponse(resp)
	if err != nil {
		t.Fatalf("AddResponse failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 3 { // system + user + assistant
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	assistantMsg := messages[2]
	if assistantMsg.Role != RoleAssistant {
		t.Errorf("Last message should be assistant")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	// Verify tool use block
	if assistantMsg.ContentBlocks[1].Type != ContentBlockTypeToolUse {
		t.Errorf("Second block should be tool_use")
	}
	if assistantMsg.ContentBlocks[1].ToolUse == nil || assistantMsg.ContentBlocks[1].ToolUse.ID != "toolu_123" {
		t.Errorf("Tool use ID not preserved")
	}
}

func TestAddResponseWithToolResult_AutoDetectAnthropic(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create Anthropic response
	text := "I'll check the weather."
	inputData, _ := json.Marshal(map[string]any{"location": "SF"})
	resp := anthropic.MessagesResponse{
		Content: []anthropic.MessageContent{
			anthropic.NewTextMessageContent(text),
			anthropic.NewToolUseMessageContent("toolu_123", "get_weather", inputData),
		},
	}

	// Should auto-detect and add both response and result
	err := conv.AddResponseWithToolResult(resp, "The weather is sunny", false)
	if err != nil {
		t.Fatalf("AddResponseWithToolResult failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 4 { // system + user + assistant + user (tool result)
		t.Errorf("Expected 4 messages, got %d", len(messages))
	}

	// Verify tool result message
	toolResultMsg := messages[3]
	if toolResultMsg.Role != RoleUser {
		t.Errorf("Tool result message should be user role")
	}

	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Errorf("Expected 1 content block in tool result, got %d", len(toolResultMsg.ContentBlocks))
	}

	if toolResultMsg.ContentBlocks[0].Type != ContentBlockTypeToolResult {
		t.Errorf("Expected tool_result block")
	}

	toolResult := toolResultMsg.ContentBlocks[0].ToolResult
	if toolResult == nil {
		t.Fatalf("Tool result is nil")
	}

	if toolResult.ToolUseID != "toolu_123" {
		t.Errorf("Tool result not linked to correct tool use ID")
	}

	if toolResult.Content != "The weather is sunny" {
		t.Errorf("Tool result content = %q, want 'The weather is sunny'", toolResult.Content)
	}
}

func TestAddResponse_AutoDetectOpenAI(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create OpenAI response
	resp := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "I'll check the weather.",
					ToolCalls: []openai.ToolCall{
						{
							ID:   "call_123",
							Type: openai.ToolTypeFunction,
							Function: openai.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"SF"}`,
							},
						},
					},
				},
			},
		},
	}

	// Should auto-detect and add without setting handler
	err := conv.AddResponse(resp)
	if err != nil {
		t.Fatalf("AddResponse failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 3 { // system + user + assistant
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	assistantMsg := messages[2]
	if assistantMsg.Role != RoleAssistant {
		t.Errorf("Last message should be assistant")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	// Verify tool use block
	if assistantMsg.ContentBlocks[1].Type != ContentBlockTypeToolUse {
		t.Errorf("Second block should be tool_use")
	}
	if assistantMsg.ContentBlocks[1].ToolUse == nil || assistantMsg.ContentBlocks[1].ToolUse.ID != "call_123" {
		t.Errorf("Tool use ID not preserved")
	}
}

func TestAddResponseWithToolResult_AutoDetectOpenAI(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create OpenAI response
	resp := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "I'll check the weather.",
					ToolCalls: []openai.ToolCall{
						{
							ID:   "call_123",
							Type: openai.ToolTypeFunction,
							Function: openai.FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"SF"}`,
							},
						},
					},
				},
			},
		},
	}

	// Should auto-detect and add both response and result
	err := conv.AddResponseWithToolResult(resp, "The weather is sunny", false)
	if err != nil {
		t.Fatalf("AddResponseWithToolResult failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 4 { // system + user + assistant + user (tool result)
		t.Errorf("Expected 4 messages, got %d", len(messages))
	}

	// Verify tool result message
	toolResultMsg := messages[3]
	if toolResultMsg.Role != RoleUser {
		t.Errorf("Tool result message should be user role")
	}

	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Errorf("Expected 1 content block in tool result, got %d", len(toolResultMsg.ContentBlocks))
	}

	if toolResultMsg.ContentBlocks[0].Type != ContentBlockTypeToolResult {
		t.Errorf("Expected tool_result block")
	}

	toolResult := toolResultMsg.ContentBlocks[0].ToolResult
	if toolResult == nil {
		t.Fatalf("Tool result is nil")
	}

	if toolResult.ToolUseID != "call_123" {
		t.Errorf("Tool result not linked to correct tool use ID")
	}

	if toolResult.Content != "The weather is sunny" {
		t.Errorf("Tool result content = %q, want 'The weather is sunny'", toolResult.Content)
	}
}

func TestAddResponse_AutoDetectGoogle(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create Google response
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "I'll check the weather."},
						{
							FunctionCall: &genai.FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"location": "SF"},
							},
						},
					},
				},
			},
		},
	}

	// Should auto-detect and add without setting handler
	err := conv.AddResponse(resp)
	if err != nil {
		t.Fatalf("AddResponse failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 3 { // system + user + assistant
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	assistantMsg := messages[2]
	if assistantMsg.Role != RoleAssistant {
		t.Errorf("Last message should be assistant")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	// Verify tool use block
	if assistantMsg.ContentBlocks[1].Type != ContentBlockTypeToolUse {
		t.Errorf("Second block should be tool_use")
	}
	if assistantMsg.ContentBlocks[1].ToolUse == nil || assistantMsg.ContentBlocks[1].ToolUse.Name != "get_weather" {
		t.Errorf("Tool use name not preserved")
	}
}

func TestAddResponseWithToolResult_AutoDetectGoogle(t *testing.T) {
	conv := NewConversation("You are helpful")
	conv.AddUserMessage("What's the weather?")

	// Create Google response
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "I'll check the weather."},
						{
							FunctionCall: &genai.FunctionCall{
								Name: "get_weather",
								Args: map[string]any{"location": "SF"},
							},
						},
					},
				},
			},
		},
	}

	// Should auto-detect and add both response and result
	err := conv.AddResponseWithToolResult(resp, "The weather is sunny", false)
	if err != nil {
		t.Fatalf("AddResponseWithToolResult failed: %v", err)
	}

	messages := conv.GetMessages()
	if len(messages) != 4 { // system + user + assistant + user (tool result)
		t.Errorf("Expected 4 messages, got %d", len(messages))
	}

	// Verify tool result message
	toolResultMsg := messages[3]
	if toolResultMsg.Role != RoleUser {
		t.Errorf("Tool result message should be user role")
	}

	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Errorf("Expected 1 content block in tool result, got %d", len(toolResultMsg.ContentBlocks))
	}

	if toolResultMsg.ContentBlocks[0].Type != ContentBlockTypeToolResult {
		t.Errorf("Expected tool_result block")
	}

	toolResult := toolResultMsg.ContentBlocks[0].ToolResult
	if toolResult == nil {
		t.Fatalf("Tool result is nil")
	}

	// Google uses function name as ID
	if toolResult.ToolUseID != "get_weather" {
		t.Errorf("Tool result not linked to correct tool use ID")
	}

	if toolResult.Content != "The weather is sunny" {
		t.Errorf("Tool result content = %q, want 'The weather is sunny'", toolResult.Content)
	}
}

func TestAddResponse_UnsupportedType(t *testing.T) {
	conv := NewConversation("You are helpful")

	// Try with an unsupported type
	err := conv.AddResponse("not a valid response")
	if err != ErrUnsupportedResponseType {
		t.Errorf("Expected ErrUnsupportedResponseType, got %v", err)
	}
}
