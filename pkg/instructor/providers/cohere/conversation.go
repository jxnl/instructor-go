package cohere

import (
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	cohere "github.com/cohere-ai/cohere-go/v2"
)

// ToCohereMessages converts unified conversation messages to Cohere format
// Note: Cohere uses Preamble for system messages in ChatRequest
func ToCohereMessages(messages []core.Message) (preamble string, chatHistory []*cohere.Message) {
	chatHistory = make([]*cohere.Message, 0)

	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			// Concatenate system messages as preamble
			if preamble != "" {
				preamble += "\n"
			}
			preamble += msg.Content
		} else {
			// Create appropriate message based on role
			var cohereMsg *cohere.Message
			switch msg.Role {
			case core.RoleUser:
				cohereMsg = &cohere.Message{
					Role: "USER",
					User: &cohere.ChatMessage{
						Message: msg.Content,
					},
				}
			case core.RoleAssistant:
				cohereMsg = &cohere.Message{
					Role: "CHATBOT",
					Chatbot: &cohere.ChatMessage{
						Message: msg.Content,
					},
				}
			default:
				// Skip unsupported roles
				continue
			}

			chatHistory = append(chatHistory, cohereMsg)
		}
	}

	return preamble, chatHistory
}

// FromCohereMessages converts Cohere messages to unified conversation format
func FromCohereMessages(preamble string, messages []*cohere.Message) []core.Message {
	result := make([]core.Message, 0)

	// Add system message if present
	if preamble != "" {
		result = append(result, core.Message{
			Role:    core.RoleSystem,
			Content: preamble,
		})
	}

	// Convert Cohere messages
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		var role core.Role
		var content string

		switch msg.Role {
		case "USER":
			role = core.RoleUser
			if msg.User != nil {
				content = msg.User.Message
			}
		case "CHATBOT":
			role = core.RoleAssistant
			if msg.Chatbot != nil {
				content = msg.Chatbot.Message
			}
		default:
			continue
		}

		result = append(result, core.Message{
			Role:    role,
			Content: content,
		})
	}

	return result
}

// ConversationToMessages is a convenience method to convert a Conversation to Cohere messages
// Returns the preamble (system prompt) and chat history that can be used to populate a ChatRequest
func ConversationToMessages(conv *core.Conversation) (preamble string, chatHistory []*cohere.Message) {
	return ToCohereMessages(conv.GetMessages())
}
