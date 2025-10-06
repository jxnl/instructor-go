package openai

import (
	"encoding/base64"

	"github.com/567-labs/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

// ResponseHandler implements core.ResponseHandler for OpenAI
type ResponseHandler struct{}

// NewResponseHandler creates a new OpenAI response handler
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// AddResponse implements core.ResponseHandler.AddResponse
func (h *ResponseHandler) AddResponse(conv *core.Conversation, response any) {
	resp, ok := response.(openai.ChatCompletionResponse)
	if !ok {
		// Try pointer type
		respPtr, ok := response.(*openai.ChatCompletionResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	AddResponseToConversation(conv, resp)
}

// AddResponseWithToolResult implements core.ResponseHandler.AddResponseWithToolResult
func (h *ResponseHandler) AddResponseWithToolResult(conv *core.Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(openai.ChatCompletionResponse)
	if !ok {
		// Try pointer type
		respPtr, ok := response.(*openai.ChatCompletionResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	AddResponseAndToolResult(conv, resp, toolResult, isError)
}

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

// AddResponseToConversation adds an OpenAI response message to the conversation
// This properly preserves tool_calls for agent loops
func AddResponseToConversation(conv *core.Conversation, resp openai.ChatCompletionResponse) {
	if len(resp.Choices) == 0 {
		return
	}

	msg := resp.Choices[0].Message
	contentBlocks := make([]core.ContentBlock, 0)

	// Add text content if present
	if msg.Content != "" {
		contentBlocks = append(contentBlocks, core.ContentBlock{
			Type: core.ContentBlockTypeText,
			Text: msg.Content,
		})
	}

	// Add tool calls if present
	for _, toolCall := range msg.ToolCalls {
		if toolCall.Type == openai.ToolTypeFunction {
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentBlockTypeToolUse,
				ToolUse: &core.ToolUseBlock{
					ID:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: []byte(toolCall.Function.Arguments),
				},
			})
		}
	}

	// Add assistant message with structured content
	if len(contentBlocks) > 0 {
		conv.AddAssistantMessageWithBlocks(contentBlocks...)
	} else if msg.Content != "" {
		// Fallback to simple text message
		conv.AddAssistantMessage(msg.Content)
	}
}

// AddResponseAndToolResult adds the response to conversation and immediately adds the tool result
// This ensures the tool result is always correctly linked to the tool use from this response
func AddResponseAndToolResult(conv *core.Conversation, resp openai.ChatCompletionResponse, toolResult string, isError bool) {
	AddResponseToConversation(conv, resp)
	conv.AddToolResult(toolResult, isError)
}
