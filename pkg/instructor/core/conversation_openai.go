package core

import (
	openai "github.com/sashabaranov/go-openai"
)

// addOpenAIResponse adds an OpenAI response to the conversation
func addOpenAIResponse(conv *Conversation, resp openai.ChatCompletionResponse) {
	if len(resp.Choices) == 0 {
		return
	}

	msg := resp.Choices[0].Message
	contentBlocks := make([]ContentBlock, 0)

	// Add text content if present
	if msg.Content != "" {
		contentBlocks = append(contentBlocks, ContentBlock{
			Type: ContentBlockTypeText,
			Text: msg.Content,
		})
	}

	// Add tool calls if present
	for _, toolCall := range msg.ToolCalls {
		if toolCall.Type == openai.ToolTypeFunction {
			contentBlocks = append(contentBlocks, ContentBlock{
				Type: ContentBlockTypeToolUse,
				ToolUse: &ToolUseBlock{
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
