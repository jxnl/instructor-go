package openai

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

func TestToOpenAIMessages(t *testing.T) {
	messages := []core.Message{
		{Role: core.RoleSystem, Content: "You are helpful"},
		{Role: core.RoleUser, Content: "Hello"},
		{Role: core.RoleAssistant, Content: "Hi there!"},
	}

	result := ToOpenAIMessages(messages)

	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	expected := []struct {
		role    string
		content string
	}{
		{"system", "You are helpful"},
		{"user", "Hello"},
		{"assistant", "Hi there!"},
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

func TestFromOpenAIMessages(t *testing.T) {
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result := FromOpenAIMessages(messages)

	if len(result) != 3 {
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

	result := ConversationToMessages(conv)

	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	if result[0].Role != "system" {
		t.Errorf("First message role = %v, want system", result[0].Role)
	}
	if result[1].Role != "user" {
		t.Errorf("Second message role = %v, want user", result[1].Role)
	}
	if result[2].Role != "assistant" {
		t.Errorf("Third message role = %v, want assistant", result[2].Role)
	}
}

func TestRoundTripConversion(t *testing.T) {
	original := []core.Message{
		{Role: core.RoleSystem, Content: "System"},
		{Role: core.RoleUser, Content: "User"},
		{Role: core.RoleAssistant, Content: "Assistant"},
	}

	openaiMsgs := ToOpenAIMessages(original)
	result := FromOpenAIMessages(openaiMsgs)

	if len(result) != len(original) {
		t.Fatalf("Length mismatch: got %d, want %d", len(result), len(original))
	}

	for i := range original {
		if result[i].Role != original[i].Role {
			t.Errorf("Message %d: role = %v, want %v", i, result[i].Role, original[i].Role)
		}
		if result[i].Content != original[i].Content {
			t.Errorf("Message %d: content = %v, want %v", i, result[i].Content, original[i].Content)
		}
	}
}
