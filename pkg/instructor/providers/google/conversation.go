package google

import (
	"encoding/json"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

// ResponseHandler implements core.ResponseHandler for Google/Gemini
type ResponseHandler struct{}

// NewResponseHandler creates a new Google response handler
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// AddResponse implements core.ResponseHandler.AddResponse
func (h *ResponseHandler) AddResponse(conv *core.Conversation, response any) {
	resp, ok := response.(*genai.GenerateContentResponse)
	if !ok {
		return
	}
	AddResponseToConversation(conv, resp)
}

// AddResponseWithToolResult implements core.ResponseHandler.AddResponseWithToolResult
func (h *ResponseHandler) AddResponseWithToolResult(conv *core.Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(*genai.GenerateContentResponse)
	if !ok {
		return
	}
	AddResponseAndToolResult(conv, resp, toolResult, isError)
}

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
		case core.RoleTool:
			// Tool responses are sent as user messages
			role = "user"
		default:
			continue
		}

		// If ContentBlocks is set, use structured content (takes precedence)
		if len(msg.ContentBlocks) > 0 {
			parts := make([]*genai.Part, 0)

			for _, block := range msg.ContentBlocks {
				switch block.Type {
				case core.ContentBlockTypeText:
					if block.Text != "" {
						parts = append(parts, &genai.Part{
							Text: block.Text,
						})
					}

				case core.ContentBlockTypeToolUse:
					if block.ToolUse != nil {
						// Convert input to map[string]any if needed
						var args map[string]any
						switch v := block.ToolUse.Input.(type) {
						case map[string]any:
							args = v
						case []byte:
							// Try to unmarshal JSON
							var m map[string]any
							if err := json.Unmarshal(v, &m); err == nil {
								args = m
							} else {
								// If unmarshal fails, wrap in a generic field
								args = map[string]any{"data": string(v)}
							}
						case string:
							// Try to unmarshal JSON string
							var m map[string]any
							if err := json.Unmarshal([]byte(v), &m); err == nil {
								args = m
							} else {
								args = map[string]any{"data": v}
							}
						default:
							// For other types, try to convert via JSON round-trip
							data, err := json.Marshal(v)
							if err == nil {
								var m map[string]any
								if err := json.Unmarshal(data, &m); err == nil {
									args = m
								}
							}
						}

						if args != nil {
							parts = append(parts, genai.NewPartFromFunctionCall(
								block.ToolUse.Name,
								args,
							))
						}
					}

				case core.ContentBlockTypeToolResult:
					if block.ToolResult != nil {
						// Create function response
						// For Google, we need to extract the function name from the tool use ID
						// Since Google uses the function name as the ID in our implementation
						response := map[string]any{
							"result": block.ToolResult.Content,
						}

						if block.ToolResult.IsError {
							response = map[string]any{
								"error": block.ToolResult.Content,
							}
						}

						parts = append(parts, genai.NewPartFromFunctionResponse(
							block.ToolResult.ToolUseID, // This is the function name in Google's case
							response,
						))
					}
				}
			}

			if len(parts) > 0 {
				result = append(result, &genai.Content{
					Role:  role,
					Parts: parts,
				})
			}
		} else {
			// Fallback to legacy fields for backward compatibility
			result = append(result, &genai.Content{
				Role: role,
				Parts: []*genai.Part{
					{Text: msg.Content},
				},
			})
		}
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

// AddResponseToConversation adds a Google/Gemini response message to the conversation
// This properly preserves function_calls for agent loops
func AddResponseToConversation(conv *core.Conversation, resp *genai.GenerateContentResponse) {
	if len(resp.Candidates) == 0 || resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return
	}

	content := resp.Candidates[0].Content
	contentBlocks := make([]core.ContentBlock, 0)

	// Process all parts in the response
	for _, part := range content.Parts {
		if part == nil {
			continue
		}

		// Add text content
		if part.Text != "" {
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentBlockTypeText,
				Text: part.Text,
			})
		}

		// Add function call (tool use)
		if part.FunctionCall != nil {
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentBlockTypeToolUse,
				ToolUse: &core.ToolUseBlock{
					ID:    part.FunctionCall.Name, // Google uses name as identifier
					Name:  part.FunctionCall.Name,
					Input: part.FunctionCall.Args,
				},
			})
		}
	}

	// Add assistant message with structured content
	if len(contentBlocks) > 0 {
		conv.AddAssistantMessageWithBlocks(contentBlocks...)
	}
}

// AddResponseAndToolResult adds the response to conversation and immediately adds the tool result
// This ensures the tool result is always correctly linked to the tool use from this response
func AddResponseAndToolResult(conv *core.Conversation, resp *genai.GenerateContentResponse, toolResult string, isError bool) {
	AddResponseToConversation(conv, resp)
	conv.AddToolResult(toolResult, isError)
}
