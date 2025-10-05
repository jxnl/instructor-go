package core

import (
	"testing"
)

func TestNewConversationVariadic(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantLen int
	}{
		{
			name:    "no arguments",
			args:    nil,
			wantLen: 0,
		},
		{
			name:    "empty string",
			args:    []string{""},
			wantLen: 0,
		},
		{
			name:    "with system prompt",
			args:    []string{"You are helpful"},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewConversation(tt.args...)
			if len(conv.messages) != tt.wantLen {
				t.Errorf("NewConversation() created %d messages, want %d", len(conv.messages), tt.wantLen)
			}
		})
	}
}

func TestAddUserMessageWithImageURLs(t *testing.T) {
	conv := NewConversation()
	conv.AddUserMessageWithImageURLs("Describe this image", "https://example.com/image.jpg")

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.Role != RoleUser {
		t.Errorf("Expected role User, got %v", msg.Role)
	}
	if msg.Content != "Describe this image" {
		t.Errorf("Expected content 'Describe this image', got %v", msg.Content)
	}
	if len(msg.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(msg.Images))
	}
	if msg.Images[0].URL != "https://example.com/image.jpg" {
		t.Errorf("Expected image URL 'https://example.com/image.jpg', got %v", msg.Images[0].URL)
	}
}

func TestAddUserMessageWithMultipleImageURLs(t *testing.T) {
	conv := NewConversation()
	conv.AddUserMessageWithImageURLs(
		"Compare these images",
		"https://example.com/image1.jpg",
		"https://example.com/image2.jpg",
	)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if len(msg.Images) != 2 {
		t.Fatalf("Expected 2 images, got %d", len(msg.Images))
	}
	if msg.Images[0].URL != "https://example.com/image1.jpg" {
		t.Errorf("Expected first image URL 'https://example.com/image1.jpg', got %v", msg.Images[0].URL)
	}
	if msg.Images[1].URL != "https://example.com/image2.jpg" {
		t.Errorf("Expected second image URL 'https://example.com/image2.jpg', got %v", msg.Images[1].URL)
	}
}

func TestAddUserMessageWithImageData(t *testing.T) {
	conv := NewConversation()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // Fake JPEG header
	conv.AddUserMessageWithImageData("Analyze this", imageData)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if len(msg.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(msg.Images))
	}
	if len(msg.Images[0].Data) != 4 {
		t.Errorf("Expected image data length 4, got %d", len(msg.Images[0].Data))
	}
}

func TestAddUserMessageWithImages(t *testing.T) {
	conv := NewConversation()
	images := []ImageContent{
		{URL: "https://example.com/image1.jpg", Detail: "high"},
		{Data: []byte{0xFF, 0xD8}, Detail: "low"},
	}
	conv.AddUserMessageWithImages("Process these", images...)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if len(msg.Images) != 2 {
		t.Fatalf("Expected 2 images, got %d", len(msg.Images))
	}
	if msg.Images[0].Detail != "high" {
		t.Errorf("Expected first image detail 'high', got %v", msg.Images[0].Detail)
	}
	if msg.Images[1].Detail != "low" {
		t.Errorf("Expected second image detail 'low', got %v", msg.Images[1].Detail)
	}
}

func TestChainedMultiModalMessages(t *testing.T) {
	conv := NewConversation("You are a vision assistant").
		AddUserMessageWithImageURLs("What's in this image?", "https://example.com/img.jpg").
		AddAssistantMessage("I see a cat").
		AddUserMessage("What color is it?")

	if conv.Length() != 4 {
		t.Errorf("Expected 4 messages, got %d", conv.Length())
	}

	messages := conv.GetMessages()
	if messages[0].Role != RoleSystem {
		t.Errorf("First message should be system")
	}
	if len(messages[1].Images) != 1 {
		t.Errorf("Second message should have 1 image")
	}
	if messages[2].Role != RoleAssistant {
		t.Errorf("Third message should be assistant")
	}
	if messages[3].Role != RoleUser {
		t.Errorf("Fourth message should be user")
	}
}
