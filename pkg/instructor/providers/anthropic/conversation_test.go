package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/567-labs/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

func TestToAnthropicMessages(t *testing.T) {
	messages := []core.Message{
		{Role: core.RoleSystem, Content: "You are helpful"},
		{Role: core.RoleUser, Content: "Hello"},
		{Role: core.RoleAssistant, Content: "Hi there!"},
	}

	system, result := ToAnthropicMessages(messages)

	if system != "You are helpful" {
		t.Errorf("System = %v, want 'You are helpful'", system)
	}

	if len(result) != 2 { // User and assistant only
		t.Errorf("Expected 2 messages (excluding system), got %d", len(result))
	}

	expected := []struct {
		role    anthropic.ChatRole
		content string
	}{
		{anthropic.RoleUser, "Hello"},
		{anthropic.RoleAssistant, "Hi there!"},
	}

	for i, want := range expected {
		if result[i].Role != want.role {
			t.Errorf("Message %d: role = %v, want %v", i, result[i].Role, want.role)
		}
		// Extract text from content
		if len(result[i].Content) > 0 {
			content := result[i].Content[0]
			if content.Type == anthropic.MessagesContentTypeText && content.Text != nil {
				if *content.Text != want.content {
					t.Errorf("Message %d: content = %v, want %v", i, *content.Text, want.content)
				}
			}
		}
	}
}

func TestFromAnthropicMessages(t *testing.T) {
	system := "You are helpful"
	text1 := "Hello"
	text2 := "Hi there!"

	messages := []anthropic.Message{
		{
			Role: anthropic.RoleUser,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(text1),
			},
		},
		{
			Role: anthropic.RoleAssistant,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(text2),
			},
		},
	}

	result := FromAnthropicMessages(system, messages)

	if len(result) != 3 { // system + 2 messages
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	expected := []struct {
		role    core.Role
		content string
	}{
		{core.RoleSystem, "You are helpful"},
		{core.RoleUser, "Hello"},
		{core.RoleAssistant, "Hi there!"},
	}

	for i, want := range expected {
		if result[i].Role != want.role {
			t.Errorf("Message %d: role = %v, want %v", i, result[i].Role, want.role)
		}
		if result[i].Content != want.content {
			t.Errorf("Message %d: content = %v, want %v", i, result[i].Content, want.content)
		}
	}
}

func TestConversationToMessages(t *testing.T) {
	conv := core.NewConversation("You are helpful")
	conv.AddUserMessage("Hello")
	conv.AddAssistantMessage("Hi!")

	system, messages := ConversationToMessages(conv)

	if system != "You are helpful" {
		t.Errorf("System = %v, want 'You are helpful'", system)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != anthropic.RoleUser {
		t.Errorf("First message role = %v, want user", messages[0].Role)
	}
	if messages[1].Role != anthropic.RoleAssistant {
		t.Errorf("Second message role = %v, want assistant", messages[1].Role)
	}
}

func TestMultipleSystemMessages(t *testing.T) {
	messages := []core.Message{
		{Role: core.RoleSystem, Content: "First system"},
		{Role: core.RoleSystem, Content: "Second system"},
		{Role: core.RoleUser, Content: "Hello"},
	}

	system, result := ToAnthropicMessages(messages)

	expected := "First system\nSecond system"
	if system != expected {
		t.Errorf("System = %v, want %v", system, expected)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 message (excluding system), got %d", len(result))
	}
}

func TestToolUseConversion(t *testing.T) {
	// Test converting a message with tool use to Anthropic format
	messages := []core.Message{
		{Role: core.RoleSystem, Content: "You are helpful"},
		{Role: core.RoleUser, Content: "Search for Go"},
		{
			Role: core.RoleAssistant,
			ContentBlocks: []core.ContentBlock{
				{
					Type: core.ContentBlockTypeText,
					Text: "I'll search for that.",
				},
				{
					Type: core.ContentBlockTypeToolUse,
					ToolUse: &core.ToolUseBlock{
						ID:    "toolu_123",
						Name:  "search",
						Input: map[string]any{"query": "Go programming"},
					},
				},
			},
		},
	}

	system, result := ToAnthropicMessages(messages)

	if system != "You are helpful" {
		t.Errorf("System = %v, want 'You are helpful'", system)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}

	// Check assistant message has 2 content blocks
	assistantMsg := result[1]
	if assistantMsg.Role != anthropic.RoleAssistant {
		t.Errorf("Message role = %v, want assistant", assistantMsg.Role)
	}

	if len(assistantMsg.Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.Content))
	}

	// Verify text content
	if assistantMsg.Content[0].Type != anthropic.MessagesContentTypeText {
		t.Errorf("First content type = %v, want text", assistantMsg.Content[0].Type)
	}
	if assistantMsg.Content[0].Text == nil || *assistantMsg.Content[0].Text != "I'll search for that." {
		t.Errorf("Text content incorrect")
	}

	// Verify tool use content
	if assistantMsg.Content[1].Type != anthropic.MessagesContentTypeToolUse {
		t.Errorf("Second content type = %v, want tool_use", assistantMsg.Content[1].Type)
	}
	if assistantMsg.Content[1].ID != "toolu_123" {
		t.Errorf("Tool use ID = %v, want 'toolu_123'", assistantMsg.Content[1].ID)
	}
	if assistantMsg.Content[1].Name != "search" {
		t.Errorf("Tool use name = %v, want 'search'", assistantMsg.Content[1].Name)
	}
}

func TestToolResultConversion(t *testing.T) {
	// Test converting a message with tool result to Anthropic format
	messages := []core.Message{
		{
			Role: core.RoleUser,
			ContentBlocks: []core.ContentBlock{
				{
					Type: core.ContentBlockTypeToolResult,
					ToolResult: &core.ToolResultBlock{
						ToolUseID: "toolu_123",
						Content:   "Search results: ...",
						IsError:   false,
					},
				},
			},
		},
	}

	_, result := ToAnthropicMessages(messages)

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	userMsg := result[0]
	if userMsg.Role != anthropic.RoleUser {
		t.Errorf("Message role = %v, want user", userMsg.Role)
	}

	if len(userMsg.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(userMsg.Content))
	}

	if userMsg.Content[0].Type != anthropic.MessagesContentTypeToolResult {
		t.Errorf("Content type = %v, want tool_result", userMsg.Content[0].Type)
	}
	if userMsg.Content[0].ToolUseID == nil || *userMsg.Content[0].ToolUseID != "toolu_123" {
		t.Errorf("Tool use ID incorrect")
	}
}

func TestFromAnthropicMessagesWithToolUse(t *testing.T) {
	// Test converting Anthropic messages with tool use back to core format
	system := "You are helpful"
	toolUseID := "toolu_123"
	text := "Let me search for that."

	inputData, _ := json.Marshal(map[string]any{"query": "Go"})
	messages := []anthropic.Message{
		{
			Role: anthropic.RoleAssistant,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(text),
				anthropic.NewToolUseMessageContent(
					toolUseID,
					"search",
					inputData,
				),
			},
		},
	}

	result := FromAnthropicMessages(system, messages)

	if len(result) != 2 { // system + assistant
		t.Errorf("Expected 2 messages, got %d", len(result))
	}

	// Check system message
	if result[0].Role != core.RoleSystem || result[0].Content != system {
		t.Errorf("System message incorrect")
	}

	// Check assistant message
	assistantMsg := result[1]
	if assistantMsg.Role != core.RoleAssistant {
		t.Errorf("Assistant role incorrect")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	// Verify text block
	if assistantMsg.ContentBlocks[0].Type != core.ContentBlockTypeText {
		t.Errorf("First block type = %v, want text", assistantMsg.ContentBlocks[0].Type)
	}
	if assistantMsg.ContentBlocks[0].Text != text {
		t.Errorf("Text = %v, want %v", assistantMsg.ContentBlocks[0].Text, text)
	}

	// Verify tool use block
	if assistantMsg.ContentBlocks[1].Type != core.ContentBlockTypeToolUse {
		t.Errorf("Second block type = %v, want tool_use", assistantMsg.ContentBlocks[1].Type)
	}
	if assistantMsg.ContentBlocks[1].ToolUse == nil {
		t.Fatalf("Tool use block is nil")
	}
	if assistantMsg.ContentBlocks[1].ToolUse.ID != toolUseID {
		t.Errorf("Tool use ID = %v, want %v", assistantMsg.ContentBlocks[1].ToolUse.ID, toolUseID)
	}
	if assistantMsg.ContentBlocks[1].ToolUse.Name != "search" {
		t.Errorf("Tool use name = %v, want 'search'", assistantMsg.ContentBlocks[1].ToolUse.Name)
	}
}

func TestConversationRoundTripWithToolUse(t *testing.T) {
	// Test that tool use survives round-trip conversion
	conv := core.NewConversation("You are helpful")
	conv.AddUserMessage("Search for Go")
	conv.AddAssistantMessageWithToolUse(
		"I'll search for that.",
		core.ToolUseBlock{
			ID:    "toolu_123",
			Name:  "search",
			Input: map[string]any{"query": "Go programming"},
		},
	)
	conv.AddToolResultMessage("toolu_123", "Search results: ...", false)

	// Convert to Anthropic format
	system, anthropicMsgs := ConversationToMessages(conv)

	// Convert back to core format
	coreMessages := FromAnthropicMessages(system, anthropicMsgs)

	// Verify we have the same number of messages
	if len(coreMessages) != len(conv.GetMessages()) {
		t.Errorf("Message count mismatch: got %d, want %d", len(coreMessages), len(conv.GetMessages()))
	}

	// Check assistant message preserved tool use
	assistantMsg := coreMessages[2]
	if assistantMsg.Role != core.RoleAssistant {
		t.Errorf("Assistant message role incorrect")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Assistant message should have 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	toolUseBlock := assistantMsg.ContentBlocks[1]
	if toolUseBlock.Type != core.ContentBlockTypeToolUse {
		t.Errorf("Expected tool_use block")
	}
	if toolUseBlock.ToolUse == nil || toolUseBlock.ToolUse.ID != "toolu_123" {
		t.Errorf("Tool use ID not preserved")
	}

	// Check tool result message
	toolResultMsg := coreMessages[3]
	if toolResultMsg.Role != core.RoleUser {
		t.Errorf("Tool result message should be user role")
	}

	if len(toolResultMsg.ContentBlocks) != 1 {
		t.Errorf("Tool result message should have 1 content block, got %d", len(toolResultMsg.ContentBlocks))
	}

	if toolResultMsg.ContentBlocks[0].Type != core.ContentBlockTypeToolResult {
		t.Errorf("Expected tool_result block")
	}
	if toolResultMsg.ContentBlocks[0].ToolResult == nil || toolResultMsg.ContentBlocks[0].ToolResult.ToolUseID != "toolu_123" {
		t.Errorf("Tool result not linked to tool use")
	}
}

func TestAddResponseToConversation(t *testing.T) {
	// Test the AddResponseToConversation helper
	conv := core.NewConversation("You are helpful")
	conv.AddUserMessage("Search for Go")

	// Simulate an Anthropic response with tool use
	text := "I'll search for that."
	inputData, _ := json.Marshal(map[string]any{"query": "Go"})
	resp := anthropic.MessagesResponse{
		Content: []anthropic.MessageContent{
			anthropic.NewTextMessageContent(text),
			anthropic.NewToolUseMessageContent(
				"toolu_123",
				"search",
				inputData,
			),
		},
	}

	// Add response to conversation
	AddResponseToConversation(conv, resp)

	messages := conv.GetMessages()
	if len(messages) != 3 { // system + user + assistant
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	assistantMsg := messages[2]
	if assistantMsg.Role != core.RoleAssistant {
		t.Errorf("Last message should be assistant")
	}

	if len(assistantMsg.ContentBlocks) != 2 {
		t.Errorf("Assistant message should have 2 content blocks, got %d", len(assistantMsg.ContentBlocks))
	}

	// Verify text preserved
	if assistantMsg.ContentBlocks[0].Type != core.ContentBlockTypeText {
		t.Errorf("First block should be text")
	}
	if assistantMsg.ContentBlocks[0].Text != text {
		t.Errorf("Text not preserved")
	}

	// Verify tool use preserved
	if assistantMsg.ContentBlocks[1].Type != core.ContentBlockTypeToolUse {
		t.Errorf("Second block should be tool_use")
	}
	if assistantMsg.ContentBlocks[1].ToolUse == nil || assistantMsg.ContentBlocks[1].ToolUse.ID != "toolu_123" {
		t.Errorf("Tool use not preserved correctly")
	}
}

func TestAddResponseAndToolResult(t *testing.T) {
	t.Run("adds response and result in one call", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Search for Go")

		// Simulate response with tool use
		text := "Let me search"
		inputData, _ := json.Marshal(map[string]any{"query": "Go"})
		resp := anthropic.MessagesResponse{
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(text),
				anthropic.NewToolUseMessageContent("tool_123", "search", inputData),
			},
		}

		// Combined API - adds response and result together
		AddResponseAndToolResult(conv, resp, "Search results here", false)

		messages := conv.GetMessages()
		if len(messages) != 3 {
			t.Fatalf("Expected 3 messages (user, assistant, tool result), got %d", len(messages))
		}

		// Verify assistant message
		assistantMsg := messages[1]
		if assistantMsg.Role != core.RoleAssistant {
			t.Error("Second message should be assistant")
		}
		if len(assistantMsg.ContentBlocks) != 2 {
			t.Error("Assistant should have text + tool use")
		}

		// Verify tool result message
		toolResultMsg := messages[2]
		if toolResultMsg.Role != core.RoleUser {
			t.Error("Third message should be user (tool result)")
		}
		if len(toolResultMsg.ContentBlocks) != 1 {
			t.Fatal("Tool result message should have 1 block")
		}
		if toolResultMsg.ContentBlocks[0].Type != core.ContentBlockTypeToolResult {
			t.Error("Should be tool_result type")
		}
		if toolResultMsg.ContentBlocks[0].ToolResult.ToolUseID != "tool_123" {
			t.Error("Tool result should be linked to tool_123")
		}
		if toolResultMsg.ContentBlocks[0].ToolResult.Content != "Search results here" {
			t.Error("Tool result content incorrect")
		}
	})

	t.Run("works with error results", func(t *testing.T) {
		conv := core.NewConversation()
		inputData, _ := json.Marshal(map[string]any{})
		resp := anthropic.MessagesResponse{
			Content: []anthropic.MessageContent{
				anthropic.NewToolUseMessageContent("tool_err", "test", inputData),
			},
		}

		AddResponseAndToolResult(conv, resp, "Error occurred", true)

		messages := conv.GetMessages()
		toolResult := messages[1].ContentBlocks[0].ToolResult

		if !toolResult.IsError {
			t.Error("IsError should be true")
		}
	})
}
