package openai

import (
	"encoding/base64"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

// ToOpenAIMessages converts unified conversation messages to OpenAI format
func ToOpenAIMessages(messages []core.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		// If the message has images, use MultiContent
		if len(msg.Images) > 0 {
			parts := make([]openai.ChatMessagePart, 0, len(msg.Images)+1)

			// Add text part if content is not empty
			if msg.Content != "" {
				parts = append(parts, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: msg.Content,
				})
			}

			// Add image parts
			for _, img := range msg.Images {
				imagePart := openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
				}

				if img.URL != "" {
					// URL-based image
					imagePart.ImageURL = &openai.ChatMessageImageURL{
						URL:    img.URL,
						Detail: openai.ImageURLDetail(img.Detail),
					}
				} else if len(img.Data) > 0 {
					// Base64-encoded image data
					base64Data := base64.StdEncoding.EncodeToString(img.Data)
					imagePart.ImageURL = &openai.ChatMessageImageURL{
						URL:    "data:image/jpeg;base64," + base64Data,
						Detail: openai.ImageURLDetail(img.Detail),
					}
				}

				parts = append(parts, imagePart)
			}

			result[i] = openai.ChatCompletionMessage{
				Role:         string(msg.Role),
				MultiContent: parts,
				Name:         msg.Name,
				ToolCallID:   msg.ToolCallID,
			}
		} else {
			// Simple text message
			result[i] = openai.ChatCompletionMessage{
				Role:       string(msg.Role),
				Content:    msg.Content,
				Name:       msg.Name,
				ToolCallID: msg.ToolCallID,
			}
		}
	}
	return result
}

// FromOpenAIMessages converts OpenAI messages to unified conversation format
func FromOpenAIMessages(messages []openai.ChatCompletionMessage) []core.Message {
	result := make([]core.Message, len(messages))
	for i, msg := range messages {
		result[i] = core.Message{
			Role:       core.Role(msg.Role),
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}
	}
	return result
}

// ConversationToMessages is a convenience method to convert a Conversation to OpenAI messages
func ConversationToMessages(conv *core.Conversation) []openai.ChatCompletionMessage {
	return ToOpenAIMessages(conv.GetMessages())
}
