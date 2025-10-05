package core

// Role represents a message role in a conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a unified message structure across all providers
type Message struct {
	Role    Role
	Content string

	// Optional fields for advanced use cases
	Name       string // For function/tool calling
	ToolCallID string // For tool responses

	// Multi-modal support
	Images []ImageContent // For vision/image messages
}

// ImageContent represents an image in a message
type ImageContent struct {
	URL    string // Image URL for URL-based images
	Data   []byte // Raw image data for base64/inline images
	Detail string // Optional: "low", "high", "auto" for image detail level
}

// Conversation manages conversation history with a provider-agnostic interface
type Conversation struct {
	messages []Message
	provider Provider
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
