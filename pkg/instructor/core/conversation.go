package core

import (
	anthropic "github.com/liushuangls/go-anthropic/v2"
	openai "github.com/sashabaranov/go-openai"
	genai "google.golang.org/genai"
)

// Type aliases for provider response types
type (
	anthropicMessagesResponse     = anthropic.MessagesResponse
	openaiChatCompletionResponse  = openai.ChatCompletionResponse
	googleGenerateContentResponse = genai.GenerateContentResponse
)

// Handler instances for each provider
type (
	anthropicResponseHandler struct{}
	openaiResponseHandler    struct{}
	googleResponseHandler    struct{}
)

// Role represents a message role in a conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ContentBlockType represents the type of content in a message
type ContentBlockType string

const (
	ContentBlockTypeText       ContentBlockType = "text"
	ContentBlockTypeImage      ContentBlockType = "image"
	ContentBlockTypeToolUse    ContentBlockType = "tool_use"
	ContentBlockTypeToolResult ContentBlockType = "tool_result"
)

// ContentBlock represents a structured content block in a message
type ContentBlock struct {
	Type ContentBlockType

	// For text content
	Text string

	// For image content
	Image *ImageContent

	// For tool use content (assistant messages)
	ToolUse *ToolUseBlock

	// For tool result content (user messages)
	ToolResult *ToolResultBlock
}

// ToolUseBlock represents a tool use request from the assistant
type ToolUseBlock struct {
	ID    string // Unique ID for this tool use
	Name  string // Name of the tool to call
	Input any    // Tool input parameters (typically map[string]any or json.RawMessage)
}

// ToolResultBlock represents the result of a tool execution
type ToolResultBlock struct {
	ToolUseID string // ID of the tool use this is responding to
	Content   string // Result content
	IsError   bool   // Whether this represents an error
}

// Message represents a unified message structure across all providers
type Message struct {
	Role    Role
	Content string // Simple text content (for backward compatibility)

	// Structured content blocks (for tool use, tool results, mixed content)
	// When ContentBlocks is set, it takes precedence over Content
	ContentBlocks []ContentBlock

	// Optional fields for advanced use cases
	Name       string // For function/tool calling
	ToolCallID string // For tool responses (OpenAI-specific)

	// Multi-modal support (for backward compatibility)
	// Deprecated: Use ContentBlocks instead
	Images []ImageContent
}

// ImageContent represents an image in a message
type ImageContent struct {
	URL    string // Image URL for URL-based images
	Data   []byte // Raw image data for base64/inline images
	Detail string // Optional: "low", "high", "auto" for image detail level
}

// ResponseHandler is an interface for provider-specific response handling
// Providers implement this to add their responses to conversations
type ResponseHandler interface {
	// AddResponse adds a provider-specific response to the conversation
	// This preserves tool_use/tool_call structures
	AddResponse(conv *Conversation, response any)

	// AddResponseWithToolResult adds a response and tool result atomically
	// This ensures the tool result is correctly linked to the tool use
	AddResponseWithToolResult(conv *Conversation, response any, toolResult string, isError bool)
}

// Conversation manages conversation history with a provider-agnostic interface
type Conversation struct {
	messages        []Message
	provider        Provider
	responseHandler ResponseHandler
}

// NewConversation creates a new conversation with an optional system prompt
func NewConversation(systemPrompt ...string) *Conversation {
	conv := &Conversation{
		messages: []Message{},
	}
	if len(systemPrompt) > 0 && systemPrompt[0] != "" {
		conv.messages = append(conv.messages, Message{
			Role:    RoleSystem,
			Content: systemPrompt[0],
		})
	}
	return conv
}

// NewConversationForProvider creates a conversation associated with a specific provider
func NewConversationForProvider(provider Provider, systemPrompt ...string) *Conversation {
	conv := NewConversation(systemPrompt...)
	conv.provider = provider
	return conv
}

// AddMessage adds a message with a specific role to the conversation
func (c *Conversation) AddMessage(role Role, content string) *Conversation {
	c.messages = append(c.messages, Message{
		Role:    role,
		Content: content,
	})
	return c
}

// AddUserMessage adds a user message to the conversation
func (c *Conversation) AddUserMessage(content string) *Conversation {
	return c.AddMessage(RoleUser, content)
}

// AddUserMessageWithImages adds a user message with images to the conversation
func (c *Conversation) AddUserMessageWithImages(content string, images ...ImageContent) *Conversation {
	c.messages = append(c.messages, Message{
		Role:    RoleUser,
		Content: content,
		Images:  images,
	})
	return c
}

// AddUserMessageWithImageURLs adds a user message with image URLs to the conversation
func (c *Conversation) AddUserMessageWithImageURLs(content string, imageURLs ...string) *Conversation {
	images := make([]ImageContent, len(imageURLs))
	for i, url := range imageURLs {
		images[i] = ImageContent{URL: url}
	}
	return c.AddUserMessageWithImages(content, images...)
}

// AddUserMessageWithImageData adds a user message with raw image data to the conversation
func (c *Conversation) AddUserMessageWithImageData(content string, imageData ...[]byte) *Conversation {
	images := make([]ImageContent, len(imageData))
	for i, data := range imageData {
		images[i] = ImageContent{Data: data}
	}
	return c.AddUserMessageWithImages(content, images...)
}

// AddAssistantMessage adds an assistant message to the conversation
func (c *Conversation) AddAssistantMessage(content string) *Conversation {
	return c.AddMessage(RoleAssistant, content)
}

// AddAssistantMessageWithBlocks adds an assistant message with structured content blocks
func (c *Conversation) AddAssistantMessageWithBlocks(blocks ...ContentBlock) *Conversation {
	// Extract text content for backward compatibility
	textContent := ""
	for _, block := range blocks {
		if block.Type == ContentBlockTypeText {
			textContent += block.Text
		}
	}

	c.messages = append(c.messages, Message{
		Role:          RoleAssistant,
		Content:       textContent, // Backward compatibility
		ContentBlocks: blocks,
	})
	return c
}

// AddAssistantMessageWithToolUse adds an assistant message containing tool use blocks
func (c *Conversation) AddAssistantMessageWithToolUse(textContent string, toolUses ...ToolUseBlock) *Conversation {
	blocks := make([]ContentBlock, 0, len(toolUses)+1)

	// Add text content if present
	if textContent != "" {
		blocks = append(blocks, ContentBlock{
			Type: ContentBlockTypeText,
			Text: textContent,
		})
	}

	// Add tool use blocks
	for _, toolUse := range toolUses {
		tu := toolUse // Create a copy
		blocks = append(blocks, ContentBlock{
			Type:    ContentBlockTypeToolUse,
			ToolUse: &tu,
		})
	}

	return c.AddAssistantMessageWithBlocks(blocks...)
}

// AddToolResultMessage adds a user message with tool result
func (c *Conversation) AddToolResultMessage(toolUseID string, content string, isError bool) *Conversation {
	c.messages = append(c.messages, Message{
		Role: RoleUser,
		ContentBlocks: []ContentBlock{
			{
				Type: ContentBlockTypeToolResult,
				ToolResult: &ToolResultBlock{
					ToolUseID: toolUseID,
					Content:   content,
					IsError:   isError,
				},
			},
		},
	})
	return c
}

// AddToolResult adds a tool result automatically linked to the last tool use
// This is a convenience method that automatically extracts the tool use ID
func (c *Conversation) AddToolResult(content string, isError bool) *Conversation {
	toolUseID := c.GetLastToolUseID()
	if toolUseID == "" {
		// No tool use found, just add as regular user message
		return c.AddUserMessage(content)
	}
	return c.AddToolResultMessage(toolUseID, content, isError)
}

// AddToolResultMessages adds a user message with multiple tool results
func (c *Conversation) AddToolResultMessages(results ...ToolResultBlock) *Conversation {
	blocks := make([]ContentBlock, len(results))
	for i, result := range results {
		r := result // Create a copy
		blocks[i] = ContentBlock{
			Type:       ContentBlockTypeToolResult,
			ToolResult: &r,
		}
	}

	c.messages = append(c.messages, Message{
		Role:          RoleUser,
		ContentBlocks: blocks,
	})
	return c
}

// GetLastToolUseID returns the ID of the most recent tool use block in the conversation
// Returns empty string if no tool use is found
func (c *Conversation) GetLastToolUseID() string {
	// Search backwards through messages for the most recent tool use
	for i := len(c.messages) - 1; i >= 0; i-- {
		msg := c.messages[i]
		if msg.Role != RoleAssistant {
			continue
		}

		// Check content blocks for tool use
		for j := len(msg.ContentBlocks) - 1; j >= 0; j-- {
			if msg.ContentBlocks[j].Type == ContentBlockTypeToolUse &&
				msg.ContentBlocks[j].ToolUse != nil {
				return msg.ContentBlocks[j].ToolUse.ID
			}
		}
	}

	return ""
}

// GetAllToolUseIDs returns all tool use IDs from the last assistant message
// Returns empty slice if no tool uses are found
func (c *Conversation) GetAllToolUseIDs() []string {
	// Search backwards for the most recent assistant message
	for i := len(c.messages) - 1; i >= 0; i-- {
		msg := c.messages[i]
		if msg.Role != RoleAssistant {
			continue
		}

		// Collect all tool use IDs from this message
		var ids []string
		for _, block := range msg.ContentBlocks {
			if block.Type == ContentBlockTypeToolUse && block.ToolUse != nil {
				ids = append(ids, block.ToolUse.ID)
			}
		}

		if len(ids) > 0 {
			return ids
		}
	}

	return nil
}

// AddSystemMessage adds a system message to the conversation
func (c *Conversation) AddSystemMessage(content string) *Conversation {
	return c.AddMessage(RoleSystem, content)
}

// GetMessages returns all messages in the conversation
func (c *Conversation) GetMessages() []Message {
	return c.messages
}

// Clear removes all messages from the conversation
func (c *Conversation) Clear() {
	c.messages = []Message{}
}

// ClearKeepingSystem removes all messages except the first system message
func (c *Conversation) ClearKeepingSystem() {
	if len(c.messages) > 0 && c.messages[0].Role == RoleSystem {
		c.messages = c.messages[:1]
	} else {
		c.messages = []Message{}
	}
}

// Length returns the number of messages in the conversation
func (c *Conversation) Length() int {
	return len(c.messages)
}

// SetProvider sets the provider for this conversation
func (c *Conversation) SetProvider(provider Provider) {
	c.provider = provider
}

// GetProvider returns the provider associated with this conversation
func (c *Conversation) GetProvider() Provider {
	return c.provider
}

// SetResponseHandler sets the response handler for this conversation
func (c *Conversation) SetResponseHandler(handler ResponseHandler) {
	c.responseHandler = handler
}

// GetResponseHandler returns the response handler associated with this conversation
func (c *Conversation) GetResponseHandler() ResponseHandler {
	return c.responseHandler
}

// AddResponse adds a provider-specific response to the conversation
// Automatically detects the provider based on response type
func (c *Conversation) AddResponse(response any) error {
	// Try to use registered handler first (for custom implementations)
	if c.responseHandler != nil {
		c.responseHandler.AddResponse(c, response)
		return nil
	}

	// Auto-detect provider from response type
	handler := detectResponseHandler(response)
	if handler == nil {
		return ErrUnsupportedResponseType
	}

	handler.AddResponse(c, response)
	return nil
}

// AddResponseWithToolResult adds a response and tool result atomically
// Automatically detects the provider based on response type
func (c *Conversation) AddResponseWithToolResult(response any, toolResult string, isError bool) error {
	// Try to use registered handler first (for custom implementations)
	if c.responseHandler != nil {
		c.responseHandler.AddResponseWithToolResult(c, response, toolResult, isError)
		return nil
	}

	// Auto-detect provider from response type
	handler := detectResponseHandler(response)
	if handler == nil {
		return ErrUnsupportedResponseType
	}

	handler.AddResponseWithToolResult(c, response, toolResult, isError)
	return nil
}

// detectResponseHandler returns the appropriate handler based on response type
func detectResponseHandler(response any) ResponseHandler {
	switch response.(type) {
	// Anthropic types
	case anthropicMessagesResponse, *anthropicMessagesResponse:
		return anthropicResponseHandler{}

	// OpenAI types
	case openaiChatCompletionResponse, *openaiChatCompletionResponse:
		return openaiResponseHandler{}

	// Google/Gemini types
	case googleGenerateContentResponse, *googleGenerateContentResponse:
		return googleResponseHandler{}

	default:
		return nil
	}
}

// ConversationError represents an error in conversation operations
type ConversationError struct {
	Message string
}

func (e *ConversationError) Error() string {
	return e.Message
}

// ErrNoResponseHandler is returned when trying to add a response without a handler (deprecated)
var ErrNoResponseHandler = &ConversationError{Message: "no response handler set - use SetResponseHandler() or provider-specific AddResponse functions"}

// ErrUnsupportedResponseType is returned when the response type is not recognized
var ErrUnsupportedResponseType = &ConversationError{Message: "unsupported response type - must be anthropic.MessagesResponse, openai.ChatCompletionResponse, or *genai.GenerateContentResponse"}

// Anthropic handler implementation
func (h anthropicResponseHandler) AddResponse(conv *Conversation, response any) {
	resp, ok := response.(anthropic.MessagesResponse)
	if !ok {
		respPtr, ok := response.(*anthropic.MessagesResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	addAnthropicResponse(conv, resp)
}

func (h anthropicResponseHandler) AddResponseWithToolResult(conv *Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(anthropic.MessagesResponse)
	if !ok {
		respPtr, ok := response.(*anthropic.MessagesResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	addAnthropicResponse(conv, resp)
	conv.AddToolResult(toolResult, isError)
}

// OpenAI handler implementation
func (h openaiResponseHandler) AddResponse(conv *Conversation, response any) {
	resp, ok := response.(openai.ChatCompletionResponse)
	if !ok {
		respPtr, ok := response.(*openai.ChatCompletionResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	addOpenAIResponse(conv, resp)
}

func (h openaiResponseHandler) AddResponseWithToolResult(conv *Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(openai.ChatCompletionResponse)
	if !ok {
		respPtr, ok := response.(*openai.ChatCompletionResponse)
		if !ok {
			return
		}
		resp = *respPtr
	}
	addOpenAIResponse(conv, resp)
	conv.AddToolResult(toolResult, isError)
}

// Google handler implementation
func (h googleResponseHandler) AddResponse(conv *Conversation, response any) {
	resp, ok := response.(*genai.GenerateContentResponse)
	if !ok {
		return
	}
	addGoogleResponse(conv, resp)
}

func (h googleResponseHandler) AddResponseWithToolResult(conv *Conversation, response any, toolResult string, isError bool) {
	resp, ok := response.(*genai.GenerateContentResponse)
	if !ok {
		return
	}
	addGoogleResponse(conv, resp)
	conv.AddToolResult(toolResult, isError)
}
