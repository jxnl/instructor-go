package core

import (
	"testing"
)

func TestGetLastToolUseID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Conversation
		expected string
	}{
		{
			name: "single tool use",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddUserMessage("Search for Go")
				conv.AddAssistantMessageWithToolUse("Let me search", ToolUseBlock{
					ID:    "tool_123",
					Name:  "search",
					Input: map[string]any{"query": "Go"},
				})
				return conv
			},
			expected: "tool_123",
		},
		{
			name: "multiple tool uses in same message",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddUserMessage("Do multiple things")
				conv.AddAssistantMessageWithToolUse("",
					ToolUseBlock{ID: "tool_1", Name: "search", Input: nil},
					ToolUseBlock{ID: "tool_2", Name: "lookup", Input: nil},
				)
				return conv
			},
			expected: "tool_2", // Should return the last one
		},
		{
			name: "tool use followed by user message",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddUserMessage("Search")
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_123",
					Name: "search",
				})
				conv.AddUserMessage("Another message")
				return conv
			},
			expected: "tool_123", // Should still find it
		},
		{
			name: "no tool use",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddUserMessage("Hello")
				conv.AddAssistantMessage("Hi there")
				return conv
			},
			expected: "", // Empty string when no tool use found
		},
		{
			name: "empty conversation",
			setup: func() *Conversation {
				return NewConversation()
			},
			expected: "",
		},
		{
			name: "tool use with tool result after",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_123",
					Name: "search",
				})
				conv.AddToolResultMessage("tool_123", "result", false)
				return conv
			},
			expected: "tool_123",
		},
		{
			name: "multiple assistant messages with tools",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_1",
					Name: "search",
				})
				conv.AddToolResultMessage("tool_1", "result", false)
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_2",
					Name: "lookup",
				})
				return conv
			},
			expected: "tool_2", // Most recent tool use
		},
		{
			name: "text and tool use mixed",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithBlocks(
					ContentBlock{Type: ContentBlockTypeText, Text: "Let me help"},
					ContentBlock{Type: ContentBlockTypeToolUse, ToolUse: &ToolUseBlock{
						ID:   "tool_456",
						Name: "helper",
					}},
					ContentBlock{Type: ContentBlockTypeText, Text: "Processing..."},
				)
				return conv
			},
			expected: "tool_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := tt.setup()
			result := conv.GetLastToolUseID()
			if result != tt.expected {
				t.Errorf("GetLastToolUseID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetAllToolUseIDs(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Conversation
		expected []string
	}{
		{
			name: "single tool use",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_1",
					Name: "search",
				})
				return conv
			},
			expected: []string{"tool_1"},
		},
		{
			name: "multiple tool uses in same message",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithToolUse("",
					ToolUseBlock{ID: "tool_1", Name: "search"},
					ToolUseBlock{ID: "tool_2", Name: "lookup"},
					ToolUseBlock{ID: "tool_3", Name: "calculate"},
				)
				return conv
			},
			expected: []string{"tool_1", "tool_2", "tool_3"},
		},
		{
			name: "no tool uses",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessage("Just text")
				return conv
			},
			expected: nil,
		},
		{
			name: "empty conversation",
			setup: func() *Conversation {
				return NewConversation()
			},
			expected: nil,
		},
		{
			name: "only returns from last assistant message",
			setup: func() *Conversation {
				conv := NewConversation()
				// Old assistant message with tool
				conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
					ID:   "tool_old",
					Name: "old",
				})
				conv.AddToolResultMessage("tool_old", "result", false)
				// New assistant message with different tools
				conv.AddAssistantMessageWithToolUse("",
					ToolUseBlock{ID: "tool_new_1", Name: "new1"},
					ToolUseBlock{ID: "tool_new_2", Name: "new2"},
				)
				return conv
			},
			expected: []string{"tool_new_1", "tool_new_2"}, // Only new ones
		},
		{
			name: "mixed content blocks",
			setup: func() *Conversation {
				conv := NewConversation()
				conv.AddAssistantMessageWithBlocks(
					ContentBlock{Type: ContentBlockTypeText, Text: "Start"},
					ContentBlock{Type: ContentBlockTypeToolUse, ToolUse: &ToolUseBlock{ID: "tool_1", Name: "a"}},
					ContentBlock{Type: ContentBlockTypeText, Text: "Middle"},
					ContentBlock{Type: ContentBlockTypeToolUse, ToolUse: &ToolUseBlock{ID: "tool_2", Name: "b"}},
				)
				return conv
			},
			expected: []string{"tool_1", "tool_2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := tt.setup()
			result := conv.GetAllToolUseIDs()

			if len(result) != len(tt.expected) {
				t.Errorf("GetAllToolUseIDs() returned %d IDs, want %d", len(result), len(tt.expected))
				return
			}

			if tt.expected == nil && result == nil {
				return // Both nil is OK
			}

			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("GetAllToolUseIDs()[%d] = %q, want %q", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestAddToolResultMessage(t *testing.T) {
	conv := NewConversation()

	// Add a tool result
	conv.AddToolResultMessage("tool_123", "Search results here", false)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.Role != RoleUser {
		t.Errorf("Expected user role, got %v", msg.Role)
	}

	if len(msg.ContentBlocks) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(msg.ContentBlocks))
	}

	block := msg.ContentBlocks[0]
	if block.Type != ContentBlockTypeToolResult {
		t.Errorf("Expected tool_result type, got %v", block.Type)
	}

	if block.ToolResult == nil {
		t.Fatal("ToolResult is nil")
	}

	if block.ToolResult.ToolUseID != "tool_123" {
		t.Errorf("ToolUseID = %q, want %q", block.ToolResult.ToolUseID, "tool_123")
	}

	if block.ToolResult.Content != "Search results here" {
		t.Errorf("Content = %q, want %q", block.ToolResult.Content, "Search results here")
	}

	if block.ToolResult.IsError != false {
		t.Errorf("IsError = %v, want false", block.ToolResult.IsError)
	}
}

func TestAddToolResultMessageError(t *testing.T) {
	conv := NewConversation()

	// Add an error tool result
	conv.AddToolResultMessage("tool_456", "Error occurred", true)

	messages := conv.GetMessages()
	msg := messages[0]

	if msg.ContentBlocks[0].ToolResult.IsError != true {
		t.Error("Expected IsError to be true")
	}
}

func TestAddToolResultMessages(t *testing.T) {
	conv := NewConversation()

	// Add multiple tool results at once
	conv.AddToolResultMessages(
		ToolResultBlock{ToolUseID: "tool_1", Content: "Result 1", IsError: false},
		ToolResultBlock{ToolUseID: "tool_2", Content: "Result 2", IsError: false},
		ToolResultBlock{ToolUseID: "tool_3", Content: "Error", IsError: true},
	)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message with multiple blocks, got %d messages", len(messages))
	}

	msg := messages[0]
	if msg.Role != RoleUser {
		t.Errorf("Expected user role, got %v", msg.Role)
	}

	if len(msg.ContentBlocks) != 3 {
		t.Fatalf("Expected 3 content blocks, got %d", len(msg.ContentBlocks))
	}

	// Verify all blocks are tool results
	for i, block := range msg.ContentBlocks {
		if block.Type != ContentBlockTypeToolResult {
			t.Errorf("Block %d: expected tool_result type, got %v", i, block.Type)
		}
	}

	// Verify specific values
	if msg.ContentBlocks[0].ToolResult.ToolUseID != "tool_1" {
		t.Error("First tool result has wrong ID")
	}
	if msg.ContentBlocks[2].ToolResult.IsError != true {
		t.Error("Third tool result should be an error")
	}
}

func TestAddAssistantMessageWithToolUse(t *testing.T) {
	conv := NewConversation()

	conv.AddAssistantMessageWithToolUse(
		"Let me search for that",
		ToolUseBlock{
			ID:    "tool_search_123",
			Name:  "search",
			Input: map[string]any{"query": "golang"},
		},
	)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.Role != RoleAssistant {
		t.Errorf("Expected assistant role, got %v", msg.Role)
	}

	if len(msg.ContentBlocks) != 2 {
		t.Fatalf("Expected 2 content blocks (text + tool), got %d", len(msg.ContentBlocks))
	}

	// Check text block
	if msg.ContentBlocks[0].Type != ContentBlockTypeText {
		t.Error("First block should be text")
	}
	if msg.ContentBlocks[0].Text != "Let me search for that" {
		t.Errorf("Text = %q, want 'Let me search for that'", msg.ContentBlocks[0].Text)
	}

	// Check tool use block
	if msg.ContentBlocks[1].Type != ContentBlockTypeToolUse {
		t.Error("Second block should be tool_use")
	}
	if msg.ContentBlocks[1].ToolUse == nil {
		t.Fatal("ToolUse is nil")
	}
	if msg.ContentBlocks[1].ToolUse.ID != "tool_search_123" {
		t.Errorf("Tool ID = %q, want 'tool_search_123'", msg.ContentBlocks[1].ToolUse.ID)
	}
}

func TestAddAssistantMessageWithToolUseNoText(t *testing.T) {
	conv := NewConversation()

	// Tool use without text is valid
	conv.AddAssistantMessageWithToolUse("", ToolUseBlock{
		ID:   "tool_1",
		Name: "test",
	})

	messages := conv.GetMessages()
	msg := messages[0]

	// Should only have tool use block, no text block
	if len(msg.ContentBlocks) != 1 {
		t.Errorf("Expected 1 content block (tool only), got %d", len(msg.ContentBlocks))
	}

	if msg.ContentBlocks[0].Type != ContentBlockTypeToolUse {
		t.Error("Block should be tool_use type")
	}
}

func TestAddAssistantMessageWithMultipleTools(t *testing.T) {
	conv := NewConversation()

	conv.AddAssistantMessageWithToolUse("Using multiple tools",
		ToolUseBlock{ID: "tool_1", Name: "search", Input: map[string]any{"q": "test"}},
		ToolUseBlock{ID: "tool_2", Name: "lookup", Input: map[string]any{"key": "value"}},
		ToolUseBlock{ID: "tool_3", Name: "calculate", Input: map[string]any{"op": "add"}},
	)

	messages := conv.GetMessages()
	msg := messages[0]

	// 1 text + 3 tools = 4 blocks
	if len(msg.ContentBlocks) != 4 {
		t.Fatalf("Expected 4 content blocks, got %d", len(msg.ContentBlocks))
	}

	// Verify order: text, then tools
	if msg.ContentBlocks[0].Type != ContentBlockTypeText {
		t.Error("First block should be text")
	}

	for i := 1; i <= 3; i++ {
		if msg.ContentBlocks[i].Type != ContentBlockTypeToolUse {
			t.Errorf("Block %d should be tool_use", i)
		}
	}
}

func TestAddToolResult(t *testing.T) {
	t.Run("automatically links to last tool use", func(t *testing.T) {
		conv := NewConversation()
		conv.AddAssistantMessageWithToolUse("Searching",
			ToolUseBlock{ID: "tool_123", Name: "search"},
		)

		// No need to pass tool use ID
		conv.AddToolResult("Results here", false)

		messages := conv.GetMessages()
		if len(messages) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(messages))
		}

		toolResultMsg := messages[1]
		if toolResultMsg.Role != RoleUser {
			t.Error("Tool result should be user role")
		}

		if len(toolResultMsg.ContentBlocks) != 1 {
			t.Fatalf("Expected 1 content block, got %d", len(toolResultMsg.ContentBlocks))
		}

		block := toolResultMsg.ContentBlocks[0]
		if block.Type != ContentBlockTypeToolResult {
			t.Error("Should be tool_result type")
		}

		if block.ToolResult.ToolUseID != "tool_123" {
			t.Errorf("ToolUseID = %q, want 'tool_123'", block.ToolResult.ToolUseID)
		}

		if block.ToolResult.Content != "Results here" {
			t.Errorf("Content = %q, want 'Results here'", block.ToolResult.Content)
		}
	})

	t.Run("works with error results", func(t *testing.T) {
		conv := NewConversation()
		conv.AddAssistantMessageWithToolUse("",
			ToolUseBlock{ID: "tool_error", Name: "test"},
		)

		conv.AddToolResult("Error occurred", true)

		messages := conv.GetMessages()
		toolResult := messages[1].ContentBlocks[0].ToolResult

		if !toolResult.IsError {
			t.Error("IsError should be true")
		}
	})

	t.Run("links to most recent tool when multiple exist", func(t *testing.T) {
		conv := NewConversation()

		// First tool use
		conv.AddAssistantMessageWithToolUse("",
			ToolUseBlock{ID: "tool_1", Name: "first"},
		)
		conv.AddToolResultMessage("tool_1", "Result 1", false)

		// Second tool use
		conv.AddAssistantMessageWithToolUse("",
			ToolUseBlock{ID: "tool_2", Name: "second"},
		)

		// Should link to tool_2 (most recent)
		conv.AddToolResult("Result 2", false)

		messages := conv.GetMessages()
		lastMsg := messages[len(messages)-1]

		if lastMsg.ContentBlocks[0].ToolResult.ToolUseID != "tool_2" {
			t.Error("Should link to most recent tool use")
		}
	})

	t.Run("falls back to user message when no tool use", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessage("Hello")
		conv.AddAssistantMessage("Hi")

		// No tool use exists, should add as regular user message
		conv.AddToolResult("Some content", false)

		messages := conv.GetMessages()
		lastMsg := messages[len(messages)-1]

		if lastMsg.Role != RoleUser {
			t.Error("Should be user role")
		}

		// Should have no content blocks, just regular content
		if len(lastMsg.ContentBlocks) > 0 {
			t.Error("Should not have content blocks when no tool use exists")
		}

		if lastMsg.Content != "Some content" {
			t.Errorf("Content = %q, want 'Some content'", lastMsg.Content)
		}
	})

	t.Run("works after provider AddResponseToConversation", func(t *testing.T) {
		conv := NewConversation()

		// Simulate provider adding response with tool use
		conv.AddAssistantMessageWithBlocks(
			ContentBlock{Type: ContentBlockTypeText, Text: "Thinking..."},
			ContentBlock{
				Type: ContentBlockTypeToolUse,
				ToolUse: &ToolUseBlock{
					ID:   "provider_tool_xyz",
					Name: "calculate",
				},
			},
		)

		// Simple API - no need to extract ID
		conv.AddToolResult("42", false)

		messages := conv.GetMessages()
		toolResult := messages[1].ContentBlocks[0].ToolResult

		if toolResult.ToolUseID != "provider_tool_xyz" {
			t.Error("Should link to provider's tool use ID")
		}
	})
}

// Test backward compatibility
func TestBackwardCompatibility(t *testing.T) {
	t.Run("simple messages still work", func(t *testing.T) {
		conv := NewConversation("System prompt")
		conv.AddUserMessage("Hello")
		conv.AddAssistantMessage("Hi there")
		conv.AddUserMessage("How are you?")

		messages := conv.GetMessages()
		if len(messages) != 4 {
			t.Fatalf("Expected 4 messages, got %d", len(messages))
		}

		// Verify content is accessible via Content field
		if messages[1].Content != "Hello" {
			t.Error("User message content not preserved")
		}
		if messages[2].Content != "Hi there" {
			t.Error("Assistant message content not preserved")
		}
	})

	t.Run("images still work", func(t *testing.T) {
		conv := NewConversation()
		conv.AddUserMessageWithImageURLs("What's this?", "http://example.com/img.jpg")

		messages := conv.GetMessages()
		if len(messages) != 1 {
			t.Fatal("Expected 1 message")
		}

		if len(messages[0].Images) != 1 {
			t.Error("Image not preserved")
		}
	})

	t.Run("conversation management still works", func(t *testing.T) {
		conv := NewConversation("System")
		conv.AddUserMessage("Test")
		conv.AddAssistantMessage("Response")

		if conv.Length() != 3 {
			t.Error("Length() not working")
		}

		conv.Clear()
		if conv.Length() != 0 {
			t.Error("Clear() not working")
		}

		conv = NewConversation("System")
		conv.AddUserMessage("Test")
		conv.ClearKeepingSystem()

		if conv.Length() != 1 {
			t.Error("ClearKeepingSystem() not working")
		}
	})
}
