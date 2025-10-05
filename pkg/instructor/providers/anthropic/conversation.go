package anthropic

import (
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// ToAnthropicMessages converts unified conversation messages to Anthropic format
// Note: Anthropic handles system messages separately in MessagesRequest.System
func ToAnthropicMessages(messages []core.Message) (system string, anthropicMessages []anthropic.Message) {
	anthropicMessages = make([]anthropic.Message, 0)

	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			// Concatenate system messages
			if system != "" {
				system += "\n"
			}
			system += msg.Content
		} else {
			// Convert role to Anthropic ChatRole
			var role anthropic.ChatRole
			switch msg.Role {
			case core.RoleUser:
				role = anthropic.RoleUser
			case core.RoleAssistant:
				role = anthropic.RoleAssistant
			default:
				// Skip unsupported roles
				continue
			}

			// Build content array
			content := make([]anthropic.MessageContent, 0)

			// Add text content if present
			if msg.Content != "" {
				content = append(content, anthropic.NewTextMessageContent(msg.Content))
			}

			// Add image content if present
			for _, img := range msg.Images {
				if img.URL != "" {
					// URL-based image (Note: Anthropic typically requires base64)
					// For URLs, we'd need to fetch and convert - for now just document this
					continue
				} else if len(img.Data) > 0 {
					// Base64-encoded image data
					source := anthropic.NewMessageContentSource(
						anthropic.MessagesContentSourceTypeBase64,
						"image/jpeg", // Default to JPEG, could be made configurable
						string(img.Data),
					)
					content = append(content, anthropic.NewImageMessageContent(source))
				}
			}

			anthropicMessages = append(anthropicMessages, anthropic.Message{
				Role:    role,
				Content: content,
			})
		}
	}

	return system, anthropicMessages
}

// FromAnthropicMessages converts Anthropic messages to unified conversation format
func FromAnthropicMessages(system string, messages []anthropic.Message) []core.Message {
	result := make([]core.Message, 0)

	// Add system message if present
	if system != "" {
		result = append(result, core.Message{
			Role:    core.RoleSystem,
			Content: system,
		})
	}

	// Convert Anthropic messages
	for _, msg := range messages {
		// Extract text content from message
		content := ""
		for _, c := range msg.Content {
			if c.Type == anthropic.MessagesContentTypeText && c.Text != nil {
				content += *c.Text
			}
		}

		var role core.Role
		switch msg.Role {
		case anthropic.RoleUser:
			role = core.RoleUser
		case anthropic.RoleAssistant:
			role = core.RoleAssistant
		}

		result = append(result, core.Message{
			Role:    role,
			Content: content,
		})
	}

	return result
}

// ConversationToMessages is a convenience method to convert a Conversation to Anthropic messages
// Returns the system prompt and messages array that can be used to populate a MessagesRequest
func ConversationToMessages(conv *core.Conversation) (system string, messages []anthropic.Message) {
	return ToAnthropicMessages(conv.GetMessages())
}
