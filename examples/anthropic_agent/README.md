# Anthropic Agent with Tool Use

This example demonstrates how to build an agent loop with Anthropic's API using the instructor-go Conversation API that properly preserves `tool_use` and `tool_result` message structures.

## The Problem

When using Anthropic's API in tool call mode, the LLM responds with messages containing structured `tool_use` content blocks. Previously, the Conversation API only stored text strings, which caused:

1. Loss of `tool_use` content blocks when adding assistant messages
2. Loss of the semantic link between tool calls and tool results
3. Agent loops where the LLM repeatedly calls the same tool because it doesn't recognize it already called it

## The Solution

The instructor-go Conversation API now supports structured content blocks:

```go
// Add assistant response with tool use preserved
anthropic_provider.AddResponseToConversation(conversation, resp)

// Add tool result linked to the tool use ID
conversation.AddToolResultMessage(toolUseID, result, false)
```

## Key Features

- **Structured Content Blocks**: Messages can contain text, images, tool_use, and tool_result blocks
- **Proper Tool Linking**: Tool results are linked to their corresponding tool_use via ID
- **Provider-Specific Serialization**: Each provider handles conversion to/from its native format
- **Backward Compatible**: Existing code continues to work with simple text messages

## Usage

```go
// Create a conversation
conv := core.NewConversation("System prompt")

// Add user message
conv.AddUserMessage("Search for Go programming")

// Make API call
system, messages := anthropic_provider.ConversationToMessages(conv)
resp, err := client.CreateMessagesUnion(ctx, anthropic.MessagesRequest{
    System:   system,
    Messages: messages,
    // ... other params
}, unionOptions)

// IMPORTANT: Add response to conversation to preserve tool_use blocks
anthropic_provider.AddResponseToConversation(conv, resp)

// Extract tool use ID from response
var toolUseID string
for _, content := range resp.Content {
    if content.Type == anthropic.MessagesContentTypeToolUse {
        toolUseID = content.ID
        break
    }
}

// Execute tool and add result
result := executeTool(action)
conv.AddToolResultMessage(toolUseID, result, false)

// Continue the loop...
```

## Running the Example

```bash
export ANTHROPIC_API_KEY=your-key-here
go run main.go
```

## How It Works

1. **Message Structure**: The core `Message` type now includes `ContentBlocks []ContentBlock` field
2. **Content Block Types**:
   - `ContentBlockTypeText`: Text content
   - `ContentBlockTypeImage`: Image content
   - `ContentBlockTypeToolUse`: Tool call from assistant
   - `ContentBlockTypeToolResult`: Tool result from user

3. **Provider Conversion**: Each provider implements conversion between core messages and provider-specific formats
   - `ToAnthropicMessages()`: Converts core messages to Anthropic's format
   - `FromAnthropicMessages()`: Converts Anthropic messages to core format
   - `AddResponseToConversation()`: Helper to add API responses to conversation

4. **Backward Compatibility**: The `Content` string field is still populated for simple text, ensuring existing code continues to work

## Architecture

```
Core Message Structure (Provider-Agnostic)
    ↓
Provider-Specific Conversion Layer
    ↓
Anthropic/OpenAI/etc Native Format
```

This design keeps provider-specific logic out of the core library while ensuring proper preservation of structured content across all providers.
