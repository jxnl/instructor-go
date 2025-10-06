package core

import (
	"encoding/json"

	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// addAnthropicResponse adds an Anthropic response to the conversation
func addAnthropicResponse(conv *Conversation, resp anthropic.MessagesResponse) {
	// Convert response content to content blocks
	contentBlocks := make([]ContentBlock, 0, len(resp.Content))

	for _, c := range resp.Content {
		switch c.Type {
		case anthropic.MessagesContentTypeText:
			if c.Text != nil && *c.Text != "" {
				contentBlocks = append(contentBlocks, ContentBlock{
					Type: ContentBlockTypeText,
					Text: *c.Text,
				})
			}

		case anthropic.MessagesContentTypeToolUse:
			if c.MessageContentToolUse != nil {
				contentBlocks = append(contentBlocks, ContentBlock{
					Type: ContentBlockTypeToolUse,
					ToolUse: &ToolUseBlock{
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

// Helper function to convert input to json.RawMessage
func toJSONRawMessage(input any) json.RawMessage {
	switch v := input.(type) {
	case json.RawMessage:
		return v
	case []byte:
		return json.RawMessage(v)
	default:
		data, _ := json.Marshal(v)
		return data
	}
}
