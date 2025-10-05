package google

import (
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

// ToGoogleContents converts unified conversation messages to Google format
func ToGoogleContents(messages []core.Message) []*genai.Content {
	result := make([]*genai.Content, 0)

	for _, msg := range messages {
		var role string
		switch msg.Role {
		case core.RoleUser:
			role = "user"
		case core.RoleAssistant:
			role = "model" // Google uses "model" instead of "assistant"
		case core.RoleSystem:
			// Google doesn't have a dedicated system role, treat as user message
			role = "user"
		default:
			continue
		}

		result = append(result, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: msg.Content},
			},
		})
	}

	return result
}

// FromGoogleContents converts Google contents to unified conversation format
func FromGoogleContents(contents []*genai.Content) []core.Message {
	result := make([]core.Message, 0)

	for _, content := range contents {
		if content == nil {
			continue
		}

		var role core.Role
		switch content.Role {
		case "user":
			role = core.RoleUser
		case "model":
			role = core.RoleAssistant
		default:
			continue
		}

		// Extract text from parts
		text := ""
		for _, part := range content.Parts {
			if part != nil && part.Text != "" {
				text += part.Text
			}
		}

		result = append(result, core.Message{
			Role:    role,
			Content: text,
		})
	}

	return result
}

// ConversationToContents is a convenience method to convert a Conversation to Google contents
func ConversationToContents(conv *core.Conversation) []*genai.Content {
	return ToGoogleContents(conv.GetMessages())
}
