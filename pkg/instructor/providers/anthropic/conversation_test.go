package anthropic

import (
	"testing"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
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
