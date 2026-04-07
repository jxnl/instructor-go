# Conversation History

The conversation history API provides a unified interface for managing multi-turn conversations across all supported LLM providers (OpenAI, Anthropic, Google, Cohere).

## Overview

Instead of manually managing provider-specific message arrays, you can use the `Conversation` type which automatically handles conversion to the appropriate format for each provider.

## Core API

### Creating a Conversation

```go
import "github.com/jxnl/instructor-go/pkg/instructor/core"

// With a system prompt
conversation := core.NewConversation("You are a helpful assistant")

// Without a system prompt (variadic - simply omit the argument)
conversation := core.NewConversation()
```

### Adding Messages

```go
// Add user message
conversation.AddUserMessage("What's the weather in SF?")

// Add assistant message
conversation.AddAssistantMessage("The weather in San Francisco is sunny and 70°F")

// Add system message
conversation.AddSystemMessage("You are now an expert in weather")

// Chain calls for convenience
conversation.
    AddUserMessage("Tell me about San Francisco").
    AddAssistantMessage("San Francisco is...").
    AddUserMessage("What about the weather?")
```

### Managing Conversations

```go
// Get all messages
messages := conversation.GetMessages()

// Get conversation length
length := conversation.Length()

// Clear all messages
conversation.Clear()

// Clear but keep the first system message
conversation.ClearKeepingSystem()
```

## Provider Integration

### OpenAI

```go
import (
    instructor_openai "github.com/jxnl/instructor-go/pkg/instructor/providers/openai"
    "github.com/sashabaranov/go-openai"
)

// Convert conversation to OpenAI messages
messages := instructor_openai.ConversationToMessages(conversation)

// Use in request
resp, err := client.CreateChatCompletion(
    ctx,
    openai.ChatCompletionRequest{
        Model:    openai.GPT4,
        Messages: messages,
    },
    &response,
)
```

### Anthropic

```go
import (
    instructor_anthropic "github.com/jxnl/instructor-go/pkg/instructor/providers/anthropic"
    "github.com/liushuangls/go-anthropic/v2"
)

// Convert conversation to Anthropic format (returns system prompt and messages)
system, messages := instructor_anthropic.ConversationToMessages(conversation)

// Create request
req := anthropic.MessagesRequest{
    Model:     anthropic.ModelClaude3Sonnet20240229,
    MaxTokens: 1024,
    System:    system,
    Messages:  messages,
}

// Use in request
resp, err := client.CreateMessages(ctx, req, &response)
```

### Google (Gemini)

```go
import instructor_google "github.com/jxnl/instructor-go/pkg/instructor/providers/google"

// Convert conversation to Google contents
contents := instructor_google.ConversationToContents(conversation)

// Use in request
resp, err := client.CreateChatCompletion(
    ctx,
    google.GoogleRequest{
        Model:    "gemini-pro",
        Contents: contents,
    },
    &response,
)
```

### Cohere

```go
import (
    instructor_cohere "github.com/jxnl/instructor-go/pkg/instructor/providers/cohere"
    "github.com/cohere-ai/cohere-go/v2"
)

// Convert conversation to Cohere format (returns preamble and chat history)
preamble, chatHistory := instructor_cohere.ConversationToMessages(conversation)

// Create request
req := &cohere.ChatRequest{
    Model:       "command-r-plus",
    Message:     "Latest user message goes here",
    ChatHistory: chatHistory,
}
if preamble != "" {
    req.Preamble = &preamble
}

// Use in request
resp, err := client.Chat(ctx, req, &response)
```

## Advanced Usage

### Provider-Agnostic Code

Write code once that works with any provider:

```go
func processWithAnyProvider(ctx context.Context, conversation *core.Conversation) error {
    // Add user message
    conversation.AddUserMessage("Analyze this data")

    // Use with any provider
    // The conversation will automatically convert to the right format

    return nil
}
```

### Conversation Management Patterns

#### Maintaining Context

```go
// Keep system prompt and recent messages
if conversation.Length() > 20 {
    messages := conversation.GetMessages()
    systemMsg := messages[0]
    recentMsgs := messages[len(messages)-10:]

    conversation.Clear()
    conversation.AddMessage(systemMsg.Role, systemMsg.Content)
    for _, msg := range recentMsgs {
        conversation.AddMessage(msg.Role, msg.Content)
    }
}
```

#### Session Management

```go
type Session struct {
    conversation *core.Conversation
    provider     core.Provider
}

func NewSession(systemPrompt string, provider core.Provider) *Session {
    return &Session{
        conversation: core.NewConversationForProvider(provider, systemPrompt),
        provider:     provider,
    }
}

func (s *Session) Reset() {
    s.conversation.ClearKeepingSystem()
}
```

## Examples

### Multi-Turn Agent

See [`examples/agent/main.go`](../examples/agent/main.go) for a complete example of an agent that:
- Maintains conversation state across multiple turns
- Uses union types to select tools
- Automatically manages message history

### Key Benefits

1. **Provider Independence**: Write code once, run on any provider
2. **Type Safety**: Unified `Message` type with consistent fields
3. **Convenience**: Simple methods for common operations
4. **Flexibility**: Access raw messages when needed
5. **No Magic**: Simple wrapper around a slice with conversion utilities

## Migration Guide

### Before (Manual Message Management)

```go
messages := []openai.ChatCompletionMessage{
    {Role: "system", Content: "You are helpful"},
    {Role: "user", Content: "Hello"},
}

// Make request
resp, _ := client.CreateChatCompletion(ctx,
    openai.ChatCompletionRequest{Model: openai.GPT4, Messages: messages},
    &result,
)

// Manually append messages
messages = append(messages, openai.ChatCompletionMessage{
    Role: "assistant", Content: result,
})
messages = append(messages, openai.ChatCompletionMessage{
    Role: "user", Content: "Next question",
})
```

### After (Conversation API)

```go
conversation := core.NewConversation("You are helpful")
conversation.AddUserMessage("Hello")

// Make request
resp, _ := client.CreateChatCompletion(ctx,
    openai.ChatCompletionRequest{
        Model:    openai.GPT4,
        Messages: instructor_openai.ConversationToMessages(conversation),
    },
    &result,
)

// Easily manage history
conversation.AddAssistantMessage(result)
conversation.AddUserMessage("Next question")
```

## API Reference

### `core.Conversation`

| Method | Description |
|--------|-------------|
| `NewConversation(systemPrompt ...string) *Conversation` | Create a new conversation with optional system prompt (variadic) |
| `NewConversationForProvider(provider Provider, systemPrompt ...string) *Conversation` | Create a conversation for a specific provider |
| `AddMessage(role Role, content string) *Conversation` | Add a message with specified role (chainable) |
| `AddUserMessage(content string) *Conversation` | Add a user message (chainable) |
| `AddAssistantMessage(content string) *Conversation` | Add an assistant message (chainable) |
| `AddSystemMessage(content string) *Conversation` | Add a system message (chainable) |
| `GetMessages() []Message` | Get all messages |
| `Length() int` | Get number of messages |
| `Clear()` | Remove all messages |
| `ClearKeepingSystem()` | Remove all messages except first system message |
| `SetProvider(provider Provider)` | Set the provider |
| `GetProvider() Provider` | Get the provider |

### `core.Message`

```go
type Message struct {
    Role       Role   // system, user, assistant, tool
    Content    string // Message content
    Name       string // Optional: for function/tool calling
    ToolCallID string // Optional: for tool responses
}
```

### `core.Role`

```go
const (
    RoleSystem    Role = "system"
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
    RoleTool      Role = "tool"
)
```
