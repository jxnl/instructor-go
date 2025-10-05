package core

import (
	"testing"
)

func TestNewConversation(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		wantLen      int
		wantRole     Role
	}{
		{
			name:         "with system prompt",
			systemPrompt: "You are a helpful assistant",
			wantLen:      1,
			wantRole:     RoleSystem,
		},
		{
			name:         "without system prompt",
			systemPrompt: "",
			wantLen:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewConversation(tt.systemPrompt)
			if len(conv.messages) != tt.wantLen {
				t.Errorf("NewConversation() created %d messages, want %d", len(conv.messages), tt.wantLen)
			}
			if tt.wantLen > 0 {
				if conv.messages[0].Role != tt.wantRole {
					t.Errorf("First message role = %v, want %v", conv.messages[0].Role, tt.wantRole)
				}
				if conv.messages[0].Content != tt.systemPrompt {
					t.Errorf("First message content = %v, want %v", conv.messages[0].Content, tt.systemPrompt)
				}
			}
		})
	}
}

func TestConversationAddMessages(t *testing.T) {
	conv := NewConversation("System prompt")

	conv.AddUserMessage("Hello")
	conv.AddAssistantMessage("Hi there!")
	conv.AddUserMessage("How are you?")

	messages := conv.GetMessages()
	if len(messages) != 4 { // system + 3 messages
		t.Errorf("Expected 4 messages, got %d", len(messages))
	}

	expected := []struct {
		role    Role
		content string
	}{
		{RoleSystem, "System prompt"},
		{RoleUser, "Hello"},
		{RoleAssistant, "Hi there!"},
		{RoleUser, "How are you?"},
	}

	for i, want := range expected {
		if messages[i].Role != want.role {
			t.Errorf("Message %d: role = %v, want %v", i, messages[i].Role, want.role)
		}
		if messages[i].Content != want.content {
			t.Errorf("Message %d: content = %v, want %v", i, messages[i].Content, want.content)
		}
	}
}

func TestConversationChaining(t *testing.T) {
	conv := NewConversation("").
		AddUserMessage("First").
		AddAssistantMessage("Second").
		AddUserMessage("Third")

	if conv.Length() != 3 {
		t.Errorf("Expected 3 messages, got %d", conv.Length())
	}
}

func TestConversationClear(t *testing.T) {
	conv := NewConversation("System prompt")
	conv.AddUserMessage("Test")
	conv.AddAssistantMessage("Response")

	conv.Clear()

	if conv.Length() != 0 {
		t.Errorf("Expected 0 messages after Clear(), got %d", conv.Length())
	}
}

func TestConversationClearKeepingSystem(t *testing.T) {
	conv := NewConversation("System prompt")
	conv.AddUserMessage("Test")
	conv.AddAssistantMessage("Response")

	conv.ClearKeepingSystem()

	if conv.Length() != 1 {
		t.Errorf("Expected 1 message after ClearKeepingSystem(), got %d", conv.Length())
	}

	messages := conv.GetMessages()
	if messages[0].Role != RoleSystem {
		t.Errorf("Expected first message to be system, got %v", messages[0].Role)
	}
}

func TestConversationClearKeepingSystemNoSystem(t *testing.T) {
	conv := NewConversation("")
	conv.AddUserMessage("Test")
	conv.AddAssistantMessage("Response")

	conv.ClearKeepingSystem()

	if conv.Length() != 0 {
		t.Errorf("Expected 0 messages when no system message exists, got %d", conv.Length())
	}
}

func TestConversationProvider(t *testing.T) {
	conv := NewConversationForProvider(ProviderOpenAI, "Test")

	if conv.GetProvider() != ProviderOpenAI {
		t.Errorf("Expected provider %v, got %v", ProviderOpenAI, conv.GetProvider())
	}

	conv.SetProvider(ProviderAnthropic)

	if conv.GetProvider() != ProviderAnthropic {
		t.Errorf("Expected provider %v after SetProvider, got %v", ProviderAnthropic, conv.GetProvider())
	}
}
