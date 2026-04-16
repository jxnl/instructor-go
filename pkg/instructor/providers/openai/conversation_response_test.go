package openai

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

func TestAddResponseToConversation_OpenAI(t *testing.T) {
	t.Run("response with text only", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Hello, how can I help?",
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(messages))
		}

		msg := messages[0]
		if msg.Role != core.RoleAssistant {
			t.Errorf("Expected assistant role, got %v", msg.Role)
		}

		if msg.Content != "Hello, how can I help?" {
			t.Errorf("Content = %q, want 'Hello, how can I help?'", msg.Content)
		}
	})

	t.Run("response with tool calls", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "Let me search for that",
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_abc123",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "search",
									Arguments: `{"query":"golang"}`,
								},
							},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(messages))
		}

		msg := messages[0]
		if len(msg.ContentBlocks) != 2 {
			t.Fatalf("Expected 2 content blocks, got %d", len(msg.ContentBlocks))
		}

		// Check text block
		if msg.ContentBlocks[0].Type != core.ContentBlockTypeText {
			t.Error("First block should be text")
		}
		if msg.ContentBlocks[0].Text != "Let me search for that" {
			t.Errorf("Text = %q", msg.ContentBlocks[0].Text)
		}

		// Check tool use block
		if msg.ContentBlocks[1].Type != core.ContentBlockTypeToolUse {
			t.Error("Second block should be tool_use")
		}
		if msg.ContentBlocks[1].ToolUse == nil {
			t.Fatal("ToolUse is nil")
		}
		if msg.ContentBlocks[1].ToolUse.ID != "call_abc123" {
			t.Errorf("Tool ID = %q, want 'call_abc123'", msg.ContentBlocks[1].ToolUse.ID)
		}
		if msg.ContentBlocks[1].ToolUse.Name != "search" {
			t.Errorf("Tool name = %q, want 'search'", msg.ContentBlocks[1].ToolUse.Name)
		}
	})

	t.Run("response with multiple tool calls", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "",
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_1",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "search",
									Arguments: `{"q":"test"}`,
								},
							},
							{
								ID:   "call_2",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "lookup",
									Arguments: `{"key":"value"}`,
								},
							},
							{
								ID:   "call_3",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "calculate",
									Arguments: `{"op":"add"}`,
								},
							},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		msg := messages[0]

		if len(msg.ContentBlocks) != 3 {
			t.Fatalf("Expected 3 content blocks, got %d", len(msg.ContentBlocks))
		}

		// All should be tool use
		for i, block := range msg.ContentBlocks {
			if block.Type != core.ContentBlockTypeToolUse {
				t.Errorf("Block %d should be tool_use", i)
			}
		}

		// Verify IDs
		ids := []string{"call_1", "call_2", "call_3"}
		for i, expectedID := range ids {
			if msg.ContentBlocks[i].ToolUse.ID != expectedID {
				t.Errorf("Tool %d ID = %q, want %q", i, msg.ContentBlocks[i].ToolUse.ID, expectedID)
			}
		}
	})

	t.Run("empty response", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{},
		}

		AddResponseToConversation(conv, resp)

		// Should not add any message
		messages := conv.GetMessages()
		if len(messages) != 1 {
			t.Errorf("Expected 1 message (user only), got %d", len(messages))
		}
	})

	t.Run("response with tool call only, no text", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "", // Empty content
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_xyz",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "calculate",
									Arguments: `{"x":5,"y":10}`,
								},
							},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		msg := messages[0]

		// Should only have tool use block, no text
		if len(msg.ContentBlocks) != 1 {
			t.Fatalf("Expected 1 content block (tool only), got %d", len(msg.ContentBlocks))
		}

		if msg.ContentBlocks[0].Type != core.ContentBlockTypeToolUse {
			t.Error("Block should be tool_use")
		}
	})

	t.Run("GetLastToolUseID after AddResponseToConversation", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role: "assistant",
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_test_123",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "test",
									Arguments: `{}`,
								},
							},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		toolID := conv.GetLastToolUseID()
		if toolID != "call_test_123" {
			t.Errorf("GetLastToolUseID() = %q, want 'call_test_123'", toolID)
		}
	})

	t.Run("tool calls with non-function type ignored", func(t *testing.T) {
		conv := core.NewConversation()
		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Role: "assistant",
						ToolCalls: []openai.ToolCall{
							{
								ID:   "call_valid",
								Type: openai.ToolTypeFunction,
								Function: openai.FunctionCall{
									Name:      "valid",
									Arguments: `{}`,
								},
							},
							{
								ID:   "call_invalid",
								Type: "some_other_type", // Not a function
								Function: openai.FunctionCall{
									Name:      "invalid",
									Arguments: `{}`,
								},
							},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		msg := messages[0]

		// Should only have one tool use block (the valid one)
		if len(msg.ContentBlocks) != 1 {
			t.Fatalf("Expected 1 content block, got %d", len(msg.ContentBlocks))
		}

		if msg.ContentBlocks[0].ToolUse.ID != "call_valid" {
			t.Error("Should only include the function-type tool call")
		}
	})
}

func TestOpenAIConversationRoundTrip(t *testing.T) {
	t.Run("simple conversation round trip", func(t *testing.T) {
		conv := core.NewConversation("You are helpful")
		conv.AddUserMessage("Hello")
		conv.AddAssistantMessage("Hi there!")

		// Convert to OpenAI format and back
		messages := ConversationToMessages(conv)
		newConv := core.NewConversation()
		for _, msg := range FromOpenAIMessages(messages) {
			newConv.AddMessage(msg.Role, msg.Content)
		}

		// Verify messages match
		if len(conv.GetMessages()) != len(newConv.GetMessages()) {
			t.Error("Message count mismatch after round trip")
		}
	})

	t.Run("conversation with tool use round trip", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Search for Go")
		conv.AddAssistantMessageWithToolUse("Let me search",
			core.ToolUseBlock{
				ID:    "call_123",
				Name:  "search",
				Input: []byte(`{"query":"Go"}`),
			},
		)
		conv.AddToolResultMessage("call_123", "Results here", false)

		messages := ConversationToMessages(conv)

		// Verify structure is preserved
		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}

		// Verify assistant message with tool call
		if messages[1].Role != "assistant" {
			t.Errorf("Message 1 role = %q, want 'assistant'", messages[1].Role)
		}
		if len(messages[1].ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(messages[1].ToolCalls))
		}
		if messages[1].ToolCalls[0].ID != "call_123" {
			t.Errorf("Tool call ID = %q, want 'call_123'", messages[1].ToolCalls[0].ID)
		}
		if messages[1].ToolCalls[0].Function.Name != "search" {
			t.Errorf("Tool name = %q, want 'search'", messages[1].ToolCalls[0].Function.Name)
		}

		// Verify tool result message
		if messages[2].Role != "tool" {
			t.Errorf("Message 2 role = %q, want 'tool'", messages[2].Role)
		}
		if messages[2].ToolCallID != "call_123" {
			t.Errorf("Tool call ID = %q, want 'call_123'", messages[2].ToolCallID)
		}
		if messages[2].Content != "Results here" {
			t.Errorf("Tool result content = %q, want 'Results here'", messages[2].Content)
		}
	})

	t.Run("tool result conversion with proper role and ID", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")
		conv.AddAssistantMessageWithToolUse("",
			core.ToolUseBlock{
				ID:    "call_abc123",
				Name:  "test_tool",
				Input: []byte(`{}`),
			},
		)
		conv.AddToolResultMessage("call_abc123", "Tool output", false)

		messages := ConversationToMessages(conv)

		// Find the tool result message
		var toolResultMsg *openai.ChatCompletionMessage
		for i, msg := range messages {
			if msg.Role == "tool" {
				toolResultMsg = &messages[i]
				break
			}
		}

		if toolResultMsg == nil {
			t.Fatal("No tool result message found")
		}

		if toolResultMsg.Role != "tool" {
			t.Errorf("Role = %q, want 'tool'", toolResultMsg.Role)
		}
		if toolResultMsg.ToolCallID != "call_abc123" {
			t.Errorf("ToolCallID = %q, want 'call_abc123'", toolResultMsg.ToolCallID)
		}
		if toolResultMsg.Content != "Tool output" {
			t.Errorf("Content = %q, want 'Tool output'", toolResultMsg.Content)
		}
	})

	t.Run("multiple tool results in one message", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")
		conv.AddAssistantMessageWithToolUse("",
			core.ToolUseBlock{ID: "call_1", Name: "tool1", Input: []byte(`{}`)},
			core.ToolUseBlock{ID: "call_2", Name: "tool2", Input: []byte(`{}`)},
		)
		conv.AddToolResultMessages(
			core.ToolResultBlock{ToolUseID: "call_1", Content: "Result 1", IsError: false},
			core.ToolResultBlock{ToolUseID: "call_2", Content: "Result 2", IsError: false},
		)

		messages := ConversationToMessages(conv)

		// Count tool result messages
		toolResultCount := 0
		for _, msg := range messages {
			if msg.Role == "tool" {
				toolResultCount++
			}
		}

		if toolResultCount != 2 {
			t.Errorf("Expected 2 tool result messages, got %d", toolResultCount)
		}

		// Verify both results are present with correct IDs
		foundCall1 := false
		foundCall2 := false
		for _, msg := range messages {
			if msg.Role == "tool" {
				if msg.ToolCallID == "call_1" && msg.Content == "Result 1" {
					foundCall1 = true
				}
				if msg.ToolCallID == "call_2" && msg.Content == "Result 2" {
					foundCall2 = true
				}
			}
		}

		if !foundCall1 {
			t.Error("Tool result for call_1 not found")
		}
		if !foundCall2 {
			t.Error("Tool result for call_2 not found")
		}
	})
}

func TestOpenAIBackwardCompatibility(t *testing.T) {
	t.Run("existing ToOpenAIMessages still works", func(t *testing.T) {
		messages := []core.Message{
			{Role: core.RoleSystem, Content: "System"},
			{Role: core.RoleUser, Content: "Hello"},
			{Role: core.RoleAssistant, Content: "Hi"},
		}

		result := ToOpenAIMessages(messages)
		if len(result) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(result))
		}

		if result[0].Content != "System" {
			t.Error("System message not preserved")
		}
	})

	t.Run("existing FromOpenAIMessages still works", func(t *testing.T) {
		messages := []openai.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
		}

		result := FromOpenAIMessages(messages)
		if len(result) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(result))
		}

		if result[0].Content != "Hello" {
			t.Error("User message not preserved")
		}
	})

	t.Run("existing ConversationToMessages still works", func(t *testing.T) {
		conv := core.NewConversation("System")
		conv.AddUserMessage("Hello")

		messages := ConversationToMessages(conv)
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})
}
