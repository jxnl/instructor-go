package core

import (
	genai "google.golang.org/genai"
)

// addGoogleResponse adds a Google/Gemini response to the conversation
func addGoogleResponse(conv *Conversation, resp *genai.GenerateContentResponse) {
	if len(resp.Candidates) == 0 || resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return
	}

	content := resp.Candidates[0].Content
	contentBlocks := make([]ContentBlock, 0)

	// Process all parts in the response
	for _, part := range content.Parts {
		if part == nil {
			continue
		}

		// Add text content
		if part.Text != "" {
			contentBlocks = append(contentBlocks, ContentBlock{
				Type: ContentBlockTypeText,
				Text: part.Text,
			})
		}

		// Add function call (tool use)
		if part.FunctionCall != nil {
			contentBlocks = append(contentBlocks, ContentBlock{
				Type: ContentBlockTypeToolUse,
				ToolUse: &ToolUseBlock{
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
