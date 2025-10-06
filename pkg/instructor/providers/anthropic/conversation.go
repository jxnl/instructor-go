package anthropic

import (
	"encoding/json"

	"github.com/567-labs/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// ResponseHandler implements core.ResponseHandler for Anthropic
type ResponseHandler struct{}

// NewResponseHandler creates a new Anthropic response handler
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// AddResponse implements core.ResponseHandler.AddResponse
func (h *ResponseHandler) AddResponse(conv *core.Conversation, response any) {
	resp, ok := response.(anthropic.MessagesResponse)
	if !ok {
		// Try pointer type
		respPtr, ok := response.(*anthropic.MessagesResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	AddResponseToConversation(conv, resp)
}

// AddResponseWithToolResult implements core.ResponseHandler.AddResponseWithToolResult
func (h *ResponseHandler) AddResponseWithToolResult(conv *core.Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(anthropic.MessagesResponse)
	if !ok {
		// Try pointer type
		respPtr, ok := response.(*anthropic.MessagesResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	AddResponseAndToolResult(conv, resp, toolResult, isError)
}

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

			// If ContentBlocks is set, use structured content (takes precedence)
			if len(msg.ContentBlocks) > 0 {
				for _, block := range msg.ContentBlocks {
					switch block.Type {
					case core.ContentBlockTypeText:
						if block.Text != "" {
							content = append(content, anthropic.NewTextMessageContent(block.Text))
						}
					case core.ContentBlockTypeImage:
						if block.Image != nil {
							if block.Image.URL != "" {
								// URL-based image (Note: Anthropic typically requires base64)
								// For URLs, we'd need to fetch and convert - for now just skip
								continue
							} else if len(block.Image.Data) > 0 {
								// Base64-encoded image data
								source := anthropic.NewMessageContentSource(
									anthropic.MessagesContentSourceTypeBase64,
									"image/jpeg", // Default to JPEG, could be made configurable
									string(block.Image.Data),
								)
								content = append(content, anthropic.NewImageMessageContent(source))
							}
						}
					case core.ContentBlockTypeToolUse:
						if block.ToolUse != nil {
							// Convert input to json.RawMessage if needed
							var inputData json.RawMessage
							switch v := block.ToolUse.Input.(type) {
							case json.RawMessage:
								inputData = v
							case []byte:
								inputData = json.RawMessage(v)
							default:
								// Marshal to JSON
								data, err := json.Marshal(v)
								if err != nil {
									continue // Skip if can't marshal
								}
								inputData = data
							}

							content = append(content, anthropic.NewToolUseMessageContent(
								block.ToolUse.ID,
								block.ToolUse.Name,
								inputData,
							))
						}
					case core.ContentBlockTypeToolResult:
						if block.ToolResult != nil {
							content = append(content, anthropic.NewToolResultMessageContent(
								block.ToolResult.ToolUseID,
								block.ToolResult.Content,
								block.ToolResult.IsError,
							))
						}
					}
				}
			} else {
				// Fallback to legacy fields for backward compatibility

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
			}

			// Only add message if it has content
			if len(content) > 0 {
				anthropicMessages = append(anthropicMessages, anthropic.Message{
					Role:    role,
					Content: content,
				})
			}
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
		var role core.Role
		switch msg.Role {
		case anthropic.RoleUser:
			role = core.RoleUser
		case anthropic.RoleAssistant:
			role = core.RoleAssistant
		}

		// Convert content blocks
		contentBlocks := make([]core.ContentBlock, 0, len(msg.Content))
		hasStructuredContent := false

		for _, c := range msg.Content {
			switch c.Type {
			case anthropic.MessagesContentTypeText:
				if c.Text != nil && *c.Text != "" {
					contentBlocks = append(contentBlocks, core.ContentBlock{
						Type: core.ContentBlockTypeText,
						Text: *c.Text,
					})
					hasStructuredContent = true
				}

			case anthropic.MessagesContentTypeToolUse:
				if c.MessageContentToolUse != nil {
					contentBlocks = append(contentBlocks, core.ContentBlock{
						Type: core.ContentBlockTypeToolUse,
						ToolUse: &core.ToolUseBlock{
							ID:    c.ID,
							Name:  c.Name,
							Input: c.Input,
						},
					})
					hasStructuredContent = true
				}

			case anthropic.MessagesContentTypeToolResult:
				if c.MessageContentToolResult != nil && c.ToolUseID != nil {
					// Extract content from tool result
					resultContent := ""
					for _, content := range c.Content {
						if content.Type == anthropic.MessagesContentTypeText && content.Text != nil {
							resultContent += *content.Text
						}
					}

					isError := false
					if c.IsError != nil {
						isError = *c.IsError
					}

					contentBlocks = append(contentBlocks, core.ContentBlock{
						Type: core.ContentBlockTypeToolResult,
						ToolResult: &core.ToolResultBlock{
							ToolUseID: *c.ToolUseID,
							Content:   resultContent,
							IsError:   isError,
						},
					})
					hasStructuredContent = true
				}

			case anthropic.MessagesContentTypeImage:
				// Handle image content (could be expanded)
				hasStructuredContent = true
			}
		}

		// Create message with structured content if available
		if hasStructuredContent {
			// Also populate the legacy Content field for backward compatibility
			// by concatenating all text blocks
			content := ""
			for _, block := range contentBlocks {
				if block.Type == core.ContentBlockTypeText {
					content += block.Text
				}
			}

			result = append(result, core.Message{
				Role:          role,
				Content:       content, // Backward compatibility
				ContentBlocks: contentBlocks,
			})
		} else {
			// Fallback to simple text content for backward compatibility
			content := ""
			for _, c := range msg.Content {
				if c.Type == anthropic.MessagesContentTypeText && c.Text != nil {
					content += *c.Text
				}
			}
			result = append(result, core.Message{
				Role:    role,
				Content: content,
			})
		}
	}

	return result
}

// ConversationToMessages is a convenience method to convert a Conversation to Anthropic messages
// Returns the system prompt and messages array that can be used to populate a MessagesRequest
func ConversationToMessages(conv *core.Conversation) (system string, messages []anthropic.Message) {
	return ToAnthropicMessages(conv.GetMessages())
}

// AddResponseToConversation adds an Anthropic response message to the conversation
// This properly preserves tool_use content blocks for agent loops
func AddResponseToConversation(conv *core.Conversation, resp anthropic.MessagesResponse) {
	// Convert response content to content blocks
	contentBlocks := make([]core.ContentBlock, 0, len(resp.Content))

	for _, c := range resp.Content {
		switch c.Type {
		case anthropic.MessagesContentTypeText:
			if c.Text != nil && *c.Text != "" {
				contentBlocks = append(contentBlocks, core.ContentBlock{
					Type: core.ContentBlockTypeText,
					Text: *c.Text,
				})
			}

		case anthropic.MessagesContentTypeToolUse:
			if c.MessageContentToolUse != nil {
				contentBlocks = append(contentBlocks, core.ContentBlock{
					Type: core.ContentBlockTypeToolUse,
					ToolUse: &core.ToolUseBlock{
						ID:    c.ID,
						Name:  c.Name,
						Input: c.Input,
					},
				})
			}
		}
	}

	// Add assistant message with structured content
	if len(contentBlocks) > 0 {
		conv.AddAssistantMessageWithBlocks(contentBlocks...)
	}
}

// AddResponseAndToolResult adds the response to conversation and immediately adds the tool result
// This ensures the tool result is always correctly linked to the tool use from this response
func AddResponseAndToolResult(conv *core.Conversation, resp anthropic.MessagesResponse, toolResult string, isError bool) {
	AddResponseToConversation(conv, resp)
	conv.AddToolResult(toolResult, isError)
}
