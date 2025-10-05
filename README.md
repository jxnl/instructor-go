# instructor-go - Structured LLM Outputs

Instructor Go is a library that makes it a breeze to work with structured outputs from large language models (LLMs).

---

[![Twitter Follow](https://img.shields.io/twitter/follow/jxnlco?style=social)](https://twitter.com/jxnlco)
[![LinkedIn Follow](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/robby-horvath/)
[![Documentation](https://img.shields.io/badge/docs-available-brightgreen)](https://go.useinstructor.com)
[![GitHub issues](https://img.shields.io/github/issues/instructor-ai/instructor-go.svg)](https://github.com/instructor-ai/instructor-go/issues)
[![Discord](https://img.shields.io/discord/1192334452110659664?label=discord)](https://discord.gg/UD9GPjbs8c)

Built on top of [`invopop/jsonschema`](https://github.com/invopop/jsonschema) and utilizing `jsonschema` Go struct tags (so no changing code logic), it provides a simple and user-friendly API to manage validation, retries, and streaming responses. Get ready to supercharge your LLM workflows!

## Install

Install the package into your code with:

```bash
go get "github.com/instructor-ai/instructor-go/pkg/instructor"
```

Import in your code:

```go
import (
	"github.com/instructor-ai/instructor-go/pkg/instructor"
)
```

## Example

As shown in the example below, by adding extra metadata to each struct field (via `jsonschema` tag) we want the model to be made aware of:

> For more information on the `jsonschema` tags available, see the [`jsonschema` godoc](https://pkg.go.dev/github.com/invopop/jsonschema?utm_source=godoc).

Running

```bash
export OPENAI_API_KEY=<Your OpenAI API Key>
go run examples/user/main.go
```

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

type Person struct {
	Name string `json:"name"          jsonschema:"title=the name,description=The name of the person,example=joe,example=lucy"`
	Age  int    `json:"age,omitempty" jsonschema:"title=the age,description=The age of the person,example=25,example=67"`
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)

	var person Person
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Extract Robby is 22 years old.",
				},
			},
		},
		&person,
	)
	_ = resp // sends back original response so no information loss from original API
	if err != nil {
		panic(err)
	}

	fmt.Printf(`
Name: %s
Age:  %d
`, person.Name, person.Age)
	/*
		Name: Robby
		Age:  22
	*/
}
```

See all examples here [`examples/README.md`](examples/README.md)

## Union Types

Union types allow LLMs to choose between multiple structured response types, making it easy to build agents that can select from a set of tools or actions. This is essential for creating flexible AI systems.

### Basic Usage

Define your variant types with a discriminator field:

```go
type EmailNotification struct {
    Type    string `json:"type" jsonschema:"const=email"`
    To      string `json:"to" jsonschema:"description=Email address"`
    Subject string `json:"subject" jsonschema:"description=Email subject"`
    Body    string `json:"body" jsonschema:"description=Email body"`
}

type SMSNotification struct {
    Type    string `json:"type" jsonschema:"const=sms"`
    Phone   string `json:"phone" jsonschema:"description=Phone number"`
    Message string `json:"message" jsonschema:"description=SMS message"`
}

type PushNotification struct {
    Type  string `json:"type" jsonschema:"const=push"`
    Title string `json:"title" jsonschema:"description=Notification title"`
    Body  string `json:"body" jsonschema:"description=Notification body"`
}
```

Use `CreateChatCompletionUnion` to let the LLM choose:

```go
result, resp, err := client.CreateChatCompletionUnion(
    ctx,
    openai.ChatCompletionRequest{
        Model: openai.GPT4oMini,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: "Send an email to john@example.com with subject 'Meeting Tomorrow'",
            },
        },
    },
    instructor.UnionOptions{
        Discriminator: "type",
        Variants:      []any{EmailNotification{}, SMSNotification{}, PushNotification{}},
    },
)

// Type switch on the result
switch notification := result.(type) {
case EmailNotification:
    fmt.Printf("Email to: %s\n", notification.To)
case SMSNotification:
    fmt.Printf("SMS to: %s\n", notification.Phone)
case PushNotification:
    fmt.Printf("Push: %s\n", notification.Title)
}
```

### Agent Pattern

Union types are perfect for building agents that choose actions:

```go
type SearchTool struct {
    Type  string `json:"type" jsonschema:"const=search"`
    Query string `json:"query" jsonschema:"description=Search query"`
}

type LookupTool struct {
    Type    string `json:"type" jsonschema:"const=lookup"`
    Keyword string `json:"keyword" jsonschema:"description=Keyword to lookup"`
}

type FinishTool struct {
    Type   string `json:"type" jsonschema:"const=finish"`
    Answer string `json:"answer" jsonschema:"description=Final answer"`
}

// Agent loop
for turn := 0; turn < maxTurns; turn++ {
    action, resp, err := client.CreateChatCompletionUnion(
        ctx,
        openai.ChatCompletionRequest{
            Model:    openai.GPT4oMini,
            Messages: messages,
        },
        instructor.UnionOptions{
            Discriminator: "type",
            Variants:      []any{SearchTool{}, LookupTool{}, FinishTool{}},
        },
    )

    switch tool := action.(type) {
    case SearchTool:
        result := executeSearch(tool.Query)
        messages = append(messages, resultMessage(result))
    case LookupTool:
        result := executeLookup(tool.Keyword)
        messages = append(messages, resultMessage(result))
    case FinishTool:
        return tool.Answer, nil
    }
}
```

See the complete agent example: [`examples/agent/main.go`](examples/agent/main.go)

### Key Features

- **Discriminator Field**: Each variant must have a discriminator field (typically `type`) with a unique `const` value
- **Type Safety**: Results are returned as concrete types for type-safe switching
- **Validation**: Automatic validation of discriminator uniqueness and field presence
- **Retry Logic**: Built-in retry handling for invalid discriminator values
- **Multiple Modes**: Works with all instructor modes (ToolCall, JSON, etc.)

## Conversation History

Instructor Go provides a unified conversation history API that works across all providers. This simplifies managing multi-turn conversations without dealing with provider-specific message formats.

### Basic Usage

```go
import (
    "github.com/instructor-ai/instructor-go/pkg/instructor/core"
    "github.com/instructor-ai/instructor-go/pkg/instructor/providers/openai"
    openaiLib "github.com/sashabaranov/go-openai"
)

// Create a conversation with a system prompt
conversation := core.NewConversation("You are a helpful assistant")

// Or without a system prompt
conversation := core.NewConversation()

// Add messages to the conversation
conversation.AddUserMessage("What's the weather in SF?")

// Convert to provider-specific format and use in requests
resp, err := client.CreateChatCompletion(
    ctx,
    openaiLib.ChatCompletionRequest{
        Model:    openaiLib.GPT4,
        Messages: openai.ConversationToMessages(conversation),
    },
    &response,
)

// Add assistant response
conversation.AddAssistantMessage(result)

// Continue the conversation
conversation.AddUserMessage("Now check Boston")
```

### Vision / Multi-Modal Support

```go
// Add a message with an image URL
conversation.AddUserMessageWithImageURLs(
    "What's in this image?",
    "https://example.com/image.jpg",
)

// Add a message with multiple images
conversation.AddUserMessageWithImageURLs(
    "Compare these images",
    "https://example.com/img1.jpg",
    "https://example.com/img2.jpg",
)

// Add a message with raw image data
imageData, _ := os.ReadFile("image.jpg")
conversation.AddUserMessageWithImageData("Analyze this", imageData)

// The provider adapter automatically converts to the correct format
messages := openai.ConversationToMessages(conversation)
```

### Multi-Provider Support

The same conversation can be used across different providers using a consistent functional API:

```go
// OpenAI
messages := openai.ConversationToMessages(conversation)

// Anthropic - returns system prompt and messages
system, messages := anthropic.ConversationToMessages(conversation)
req := anthropic.MessagesRequest{
    Model:    anthropic.ModelClaude3Haiku20240307,
    System:   system,
    Messages: messages,
}

// Google
contents := google.ConversationToContents(conversation)

// Cohere - returns preamble and chat history
preamble, chatHistory := cohere.ConversationToMessages(conversation)
req := cohere.ChatRequest{
    Model:       "command-r-plus",
    Preamble:    &preamble,
    ChatHistory: chatHistory,
}
```

### Conversation Management

```go
// Get all messages
messages := conversation.GetMessages()

// Get conversation length
length := conversation.Length()

// Clear all messages
conversation.Clear()

// Clear but keep system message
conversation.ClearKeepingSystem()
```

See the complete agent example: [`examples/agent/main.go`](examples/agent/main.go)

## Providers

Instructor Go supports the following LLM provider APIs:
- [OpenAI](https://github.com/sashabaranov/go-openai)
- [Anthropic](https://github.com/liushuangls/go-anthropic)
- [Cohere](github.com/cohere-ai/cohere-go)
- [Google](github.com/googleapis/go-genai)

### Usage (token counts)

These provider APIs include usage data (input and output token counts) in their responses, which Instructor Go captures and returns in the response object.

Usage is summed for retries. If multiple requests are needed to get a valid response, the usage from all requests is summed and returned. Even if Instructor fails to get a valid response after the maximum number of retries, the usage sum from all attempts is still returned.

### How to view usage data

<details>
<summary>Usage counting with OpenAI</summary>

```go
resp, err := client.CreateChatCompletion(
    ctx,
    openai.ChatCompletionRequest{
        Model: openai.GPT4o,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: "Extract Robby is 22 years old.",
            },
        },
    },
    &person,
)

fmt.Printf("Input tokens: %d\n", resp.Usage.PromptTokens)
fmt.Printf("Output tokens: %d\n", resp.Usage.CompletionTokens)
fmt.Printf("Total tokens: %d\n", resp.Usage.TotalTokens)
```

</details>

<details>
<summary>Usage counting with Anthropic</summary>

```go
resp, err := client.CreateMessages(
    ctx,
    anthropic.MessagesRequest{
        Model: anthropic.ModelClaude3Haiku20240307,
        Messages: []anthropic.Message{
            anthropic.NewUserTextMessage("Classify the following support ticket: My account is locked and I can't access my billing info."),
        },
		MaxTokens: 500,
    },
    &prediction,
)

fmt.Printf("Input tokens: %d\n", resp.Usage.InputTokens)
fmt.Printf("Output tokens: %d\n", resp.Usage.OutputTokens)
```

</details>

<details>
<summary>Usage counting with Cohere</summary>

```go
resp, err := client.Chat(
    ctx,
    &cohere.ChatRequest{
        Model: "command-r-plus",
        Message: "Tell me about the history of artificial intelligence up to year 2000",
        MaxTokens: 2500,
    },
    &historicalFact,
)

fmt.Printf("Input tokens: %d\n", int(*resp.Meta.Tokens.InputTokens))
fmt.Printf("Output tokens: %d\n", int(*resp.Meta.Tokens.OutputTokens))
```

</details>