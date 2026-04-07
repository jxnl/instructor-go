package google

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

func TestAddResponseToConversation_Google(t *testing.T) {
	t.Run("response with text only", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Hello, how can I help you today?"},
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
		if msg.Role != core.RoleAssistant {
			t.Errorf("Expected assistant role, got %v", msg.Role)
		}

		if msg.Content != "Hello, how can I help you today?" {
			t.Errorf("Content = %q", msg.Content)
		}
	})

	t.Run("response with function call", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Let me search for that"},
							{FunctionCall: &genai.FunctionCall{
								Name: "search",
								Args: map[string]any{"query": "golang"},
							}},
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

		// Check function call block
		if msg.ContentBlocks[1].Type != core.ContentBlockTypeToolUse {
			t.Error("Second block should be tool_use")
		}
		if msg.ContentBlocks[1].ToolUse == nil {
			t.Fatal("ToolUse is nil")
		}
		if msg.ContentBlocks[1].ToolUse.Name != "search" {
			t.Errorf("Function name = %q, want 'search'", msg.ContentBlocks[1].ToolUse.Name)
		}
		// Google uses name as ID
		if msg.ContentBlocks[1].ToolUse.ID != "search" {
			t.Errorf("Function ID = %q, want 'search'", msg.ContentBlocks[1].ToolUse.ID)
		}
	})

	t.Run("response with multiple function calls", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{FunctionCall: &genai.FunctionCall{
								Name: "search",
								Args: map[string]any{"q": "test"},
							}},
							{FunctionCall: &genai.FunctionCall{
								Name: "lookup",
								Args: map[string]any{"key": "value"},
							}},
							{FunctionCall: &genai.FunctionCall{
								Name: "calculate",
								Args: map[string]any{"op": "add"},
							}},
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

		// Verify names
		names := []string{"search", "lookup", "calculate"}
		for i, expectedName := range names {
			if msg.ContentBlocks[i].ToolUse.Name != expectedName {
				t.Errorf("Tool %d name = %q, want %q", i, msg.ContentBlocks[i].ToolUse.Name, expectedName)
			}
		}
	})

	t.Run("empty response", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		AddResponseToConversation(conv, resp)

		// Should not add any message
		messages := conv.GetMessages()
		if len(messages) != 1 {
			t.Errorf("Expected 1 message (user only), got %d", len(messages))
		}
	})

	t.Run("nil candidate", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{nil},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(messages))
		}
	})

	t.Run("nil content", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{Content: nil},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		if len(messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(messages))
		}
	})

	t.Run("response with function call only, no text", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{FunctionCall: &genai.FunctionCall{
								Name: "calculate",
								Args: map[string]any{"x": 5, "y": 10},
							}},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		msg := messages[0]

		// Should only have function call block, no text
		if len(msg.ContentBlocks) != 1 {
			t.Fatalf("Expected 1 content block (function only), got %d", len(msg.ContentBlocks))
		}

		if msg.ContentBlocks[0].Type != core.ContentBlockTypeToolUse {
			t.Error("Block should be tool_use")
		}
	})

	t.Run("GetLastToolUseID after AddResponseToConversation", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{FunctionCall: &genai.FunctionCall{
								Name: "test_function",
								Args: map[string]any{},
							}},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		toolID := conv.GetLastToolUseID()
		// Google uses function name as ID
		if toolID != "test_function" {
			t.Errorf("GetLastToolUseID() = %q, want 'test_function'", toolID)
		}
	})

	t.Run("mixed text and function calls", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Let me help you with that."},
							{FunctionCall: &genai.FunctionCall{
								Name: "helper",
								Args: map[string]any{"action": "assist"},
							}},
							{Text: " Processing..."},
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

		// Verify order and types
		if msg.ContentBlocks[0].Type != core.ContentBlockTypeText {
			t.Error("First block should be text")
		}
		if msg.ContentBlocks[1].Type != core.ContentBlockTypeToolUse {
			t.Error("Second block should be tool_use")
		}
		if msg.ContentBlocks[2].Type != core.ContentBlockTypeText {
			t.Error("Third block should be text")
		}

		// Verify content is concatenated for backward compatibility
		expectedContent := "Let me help you with that. Processing..."
		if msg.Content != expectedContent {
			t.Errorf("Content = %q, want %q", msg.Content, expectedContent)
		}
	})

	t.Run("nil parts in response", func(t *testing.T) {
		conv := core.NewConversation()
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Hello"},
							nil, // nil part
							{Text: "World"},
						},
					},
				},
			},
		}

		AddResponseToConversation(conv, resp)

		messages := conv.GetMessages()
		msg := messages[0]

		// Should skip nil part
		if len(msg.ContentBlocks) != 2 {
			t.Fatalf("Expected 2 content blocks (skipping nil), got %d", len(msg.ContentBlocks))
		}

		if msg.Content != "HelloWorld" {
			t.Errorf("Content = %q, want 'HelloWorld'", msg.Content)
		}
	})
}

func TestGoogleConversationRoundTrip(t *testing.T) {
	t.Run("simple conversation round trip", func(t *testing.T) {
		conv := core.NewConversation("You are helpful")
		conv.AddUserMessage("Hello")
		conv.AddAssistantMessage("Hi there!")

		// Convert to Google format
		contents := ConversationToContents(conv)

		// Verify structure
		if len(contents) != 3 {
			t.Errorf("Expected 3 contents, got %d", len(contents))
		}
	})

	t.Run("conversation with function call round trip", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Search for Go")
		conv.AddAssistantMessageWithToolUse("Let me search",
			core.ToolUseBlock{
				ID:    "search",
				Name:  "search",
				Input: map[string]any{"query": "Go"},
			},
		)

		contents := ConversationToContents(conv)

		// Verify structure is preserved
		if len(contents) != 2 {
			t.Errorf("Expected 2 contents, got %d", len(contents))
		}

		// Verify assistant message with function call
		if contents[1].Role != "model" {
			t.Errorf("Content 1 role = %q, want 'model'", contents[1].Role)
		}
		if len(contents[1].Parts) != 2 { // text + function call
			t.Fatalf("Expected 2 parts, got %d", len(contents[1].Parts))
		}
		if contents[1].Parts[0].Text != "Let me search" {
			t.Errorf("Part 0 text = %q, want 'Let me search'", contents[1].Parts[0].Text)
		}
		if contents[1].Parts[1].FunctionCall == nil {
			t.Fatal("Part 1 should have FunctionCall")
		}
		if contents[1].Parts[1].FunctionCall.Name != "search" {
			t.Errorf("Function name = %q, want 'search'", contents[1].Parts[1].FunctionCall.Name)
		}
	})

	t.Run("conversation with tool result conversion", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")
		conv.AddAssistantMessageWithToolUse("",
			core.ToolUseBlock{
				ID:    "search",
				Name:  "search",
				Input: map[string]any{"query": "Go"},
			},
		)
		conv.AddToolResultMessage("search", "Results here", false)

		contents := ConversationToContents(conv)

		// Verify we have 3 contents: user, model (with function call), user (with function response)
		if len(contents) != 3 {
			t.Fatalf("Expected 3 contents, got %d", len(contents))
		}

		// Verify function response content
		responseContent := contents[2]
		if responseContent.Role != "user" {
			t.Errorf("Function response role = %q, want 'user'", responseContent.Role)
		}
		if len(responseContent.Parts) != 1 {
			t.Fatalf("Expected 1 part in response, got %d", len(responseContent.Parts))
		}
		if responseContent.Parts[0].FunctionResponse == nil {
			t.Fatal("Part should have FunctionResponse")
		}
		if responseContent.Parts[0].FunctionResponse.Name != "search" {
			t.Errorf("Function response name = %q, want 'search'", responseContent.Parts[0].FunctionResponse.Name)
		}
		if responseContent.Parts[0].FunctionResponse.Response["result"] != "Results here" {
			t.Errorf("Function response result = %v, want 'Results here'", responseContent.Parts[0].FunctionResponse.Response["result"])
		}
	})

	t.Run("tool result with error flag", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")
		conv.AddAssistantMessageWithToolUse("",
			core.ToolUseBlock{
				ID:    "calculate",
				Name:  "calculate",
				Input: map[string]any{"op": "divide", "a": 10, "b": 0},
			},
		)
		conv.AddToolResultMessage("calculate", "Division by zero error", true)

		contents := ConversationToContents(conv)

		// Find the function response
		var funcResp *genai.FunctionResponse
		for _, content := range contents {
			for _, part := range content.Parts {
				if part.FunctionResponse != nil {
					funcResp = part.FunctionResponse
					break
				}
			}
		}

		if funcResp == nil {
			t.Fatal("No function response found")
		}

		// For errors, response should have "error" key
		if funcResp.Response["error"] != "Division by zero error" {
			t.Errorf("Function response error = %v, want 'Division by zero error'", funcResp.Response["error"])
		}
	})

	t.Run("multiple tool results in sequence", func(t *testing.T) {
		conv := core.NewConversation()
		conv.AddUserMessage("Test")
		conv.AddAssistantMessageWithToolUse("",
			core.ToolUseBlock{ID: "search", Name: "search", Input: map[string]any{"q": "test"}},
			core.ToolUseBlock{ID: "lookup", Name: "lookup", Input: map[string]any{"key": "value"}},
		)
		conv.AddToolResultMessages(
			core.ToolResultBlock{ToolUseID: "search", Content: "Result 1", IsError: false},
			core.ToolResultBlock{ToolUseID: "lookup", Content: "Result 2", IsError: false},
		)

		contents := ConversationToContents(conv)

		// Count function responses
		funcResponseCount := 0
		for _, content := range contents {
			for _, part := range content.Parts {
				if part.FunctionResponse != nil {
					funcResponseCount++
				}
			}
		}

		if funcResponseCount != 2 {
			t.Errorf("Expected 2 function responses, got %d", funcResponseCount)
		}
	})
}

func TestGoogleBackwardCompatibility(t *testing.T) {
	t.Run("existing ToGoogleContents still works", func(t *testing.T) {
		messages := []core.Message{
			{Role: core.RoleSystem, Content: "System"},
			{Role: core.RoleUser, Content: "Hello"},
			{Role: core.RoleAssistant, Content: "Hi"},
		}

		result := ToGoogleContents(messages)
		if len(result) != 3 {
			t.Errorf("Expected 3 contents, got %d", len(result))
		}
	})

	t.Run("existing FromGoogleContents still works", func(t *testing.T) {
		contents := []*genai.Content{
			{Role: "user", Parts: []*genai.Part{{Text: "Hello"}}},
			{Role: "model", Parts: []*genai.Part{{Text: "Hi"}}},
		}

		result := FromGoogleContents(contents)
		if len(result) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(result))
		}

		if result[0].Content != "Hello" {
			t.Error("User message not preserved")
		}
	})

	t.Run("existing ConversationToContents still works", func(t *testing.T) {
		conv := core.NewConversation("System")
		conv.AddUserMessage("Hello")

		contents := ConversationToContents(conv)
		if len(contents) != 2 {
			t.Errorf("Expected 2 contents, got %d", len(contents))
		}
	})
}
