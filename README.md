# instructor-go: Structured LLM Outputs

Get reliable structured data from any LLM. Built on `jsonschema` for validation, type safety, and compile-time guarantees.

```go
import (
    "github.com/567-labs/instructor-go/pkg/instructor"
    "github.com/sashabaranov/go-openai"
)

// Define what you want
type User struct {
    Name string `json:"name" jsonschema:"description=The person's name"`
    Age  int    `json:"age"  jsonschema:"description=The person's age"`
}

// Extract it from natural language
client := instructor.FromOpenAI(openai.NewClient(apiKey))
var user User
_, err := client.CreateChatCompletion(
    ctx,
    openai.ChatCompletionRequest{
        Model: openai.GPT4o,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleUser, Content: "John is 25 years old"},
        },
    },
    &user,
)

fmt.Println(user) // User{Name: "John", Age: 25}
```

**That's it.** No JSON parsing, no error handling, no retries. Just define a struct and get validated data.

[![Documentation](https://pkg.go.dev/badge/github.com/instructor-ai/instructor-go/pkg/instructor.svg)](https://pkg.go.dev/github.com/instructor-ai/instructor-go/pkg/instructor)
[![GitHub Stars](https://img.shields.io/github/stars/567-labs/instructor-go?style=flat-square)](https://github.com/567-labs/instructor-go)
[![Discord](https://img.shields.io/discord/1192334452110659664?label=discord)](https://discord.gg/UD9GPjbs8c)

***

## Why Instructor Go?

Getting structured data from LLMs is hard. You need to:

1. Write complex JSON schemas
2. Handle validation errors
3. Retry failed extractions
4. Parse unstructured responses
5. Deal with different provider APIs

**Instructor Go handles all of this with one simple interface:**

<table>
<tr>
<td><b>Without Instructor</b></td>
<td><b>With Instructor</b></td>
</tr>
<tr>
<td>

```go
resp, err := client.CreateChatCompletion(
    ctx,
    openai.ChatCompletionRequest{
        Model: openai.GPT4,
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: "..."},
        },
        Tools: []openai.Tool{
            {
                Type: openai.ToolTypeFunction,
                Function: &openai.FunctionDefinition{
                    Name: "extract_user",
                    Parameters: json.RawMessage(`{
                        "type": "object",
                        "properties": {
                            "name": {"type": "string"},
                            "age": {"type": "integer"}
                        }
                    }`),
                },
            },
        },
    },
)

// Parse response
toolCall := resp.Choices[0].Message.ToolCalls[0]
var userData map[string]interface{}
json.Unmarshal([]byte(toolCall.Function.Arguments), &userData)

// Validate manually
if _, ok := userData["name"]; !ok {
    // Handle error...
}
```

</td>
<td>

```go
client := instructor.FromOpenAI(
    openai.NewClient(apiKey),
)

var user User
_, err := client.CreateChatCompletion(
    ctx,
    openai.ChatCompletionRequest{
        Model: openai.GPT4,
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: "..."},
        },
    },
    &user,
)

// That's it! user is validated and typed
```

</td>
</tr>
</table>

## Install in seconds

```bash
go get github.com/567-labs/instructor-go/pkg/instructor
```

## Works with every major provider

Use the same code with any LLM provider:

```go
import (
    "github.com/567-labs/instructor-go/pkg/instructor"
    instructor_openai "github.com/567-labs/instructor-go/pkg/instructor/providers/openai"
    instructor_anthropic "github.com/567-labs/instructor-go/pkg/instructor/providers/anthropic"
    instructor_google "github.com/567-labs/instructor-go/pkg/instructor/providers/google"
    instructor_cohere "github.com/567-labs/instructor-go/pkg/instructor/providers/cohere"
)

// OpenAI
client := instructor.FromOpenAI(openai.NewClient(apiKey))

// Anthropic
client := instructor.FromAnthropic(anthropic.NewClient(apiKey))

// Google Gemini
client := instructor.FromGoogle(google.NewClient(apiKey))

// Cohere
client := instructor.FromCohere(cohere.NewClient(apiKey))

// All use the same API pattern!
var result YourStruct
_, err := client.CreateChatCompletion(ctx, request, &result)
```

## Production-ready features

### Automatic retries with validation

Failed validations are automatically retried with descriptive error messages:

```go
client := instructor.FromOpenAI(
    openai.NewClient(apiKey),
    instructor.WithMaxRetries(3),
    instructor.WithMode(instructor.ModeToolCall),
)

type User struct {
    Name string `json:"name" jsonschema:"description=The person's name,minLength=1"`
    Age  int    `json:"age"  jsonschema:"description=The person's age,minimum=0,maximum=150"`
}

// Instructor automatically retries when validation fails
var user User
_, err := client.CreateChatCompletion(ctx, request, &user)
```

### Debug mode for troubleshooting

Hit "max retry attempts" errors? Enable debug logging to see exactly what's happening:

```go
client := instructor.FromOpenAI(
    openai.NewClient(apiKey),
    instructor.WithLogging("debug"), // 🔍 See all retry attempts
)
```

**Output shows:**

* Each retry attempt with full context
* Response previews on failures
* Token usage across retries
* Exact JSON/validation errors

**Common formats:**

```go
instructor.WithLogging("debug")       // Development - see everything
instructor.WithLogging("json")        // Production - structured logs
instructor.WithLogging("json:error")  // Production - only errors
```

### Streaming support

Stream partial objects as they're generated:

```go
import "github.com/567-labs/instructor-go/pkg/instructor"

type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

schema, _ := instructor.GetSchema(&User{})
client := instructor.FromOpenAI(openai.NewClient(apiKey))

stream, err := client.CreateChatCompletionStream(ctx, request, schema)
defer stream.Close()

for stream.Next() {
    var partial User
    if err := stream.Scan(&partial); err == nil {
        fmt.Printf("Partial: %+v\n", partial)
        // User{Name: "", Age: 0}
        // User{Name: "John", Age: 0}
        // User{Name: "John", Age: 25}
    }
}
```

### Nested and complex structures

Extract complex, nested data structures with full type safety:

```go
type Address struct {
    Street  string `json:"street"  jsonschema:"description=Street address"`
    City    string `json:"city"    jsonschema:"description=City name"`
    Country string `json:"country" jsonschema:"description=Country name"`
}

type User struct {
    Name      string    `json:"name"      jsonschema:"description=User's full name"`
    Age       int       `json:"age"       jsonschema:"description=User's age"`
    Addresses []Address `json:"addresses" jsonschema:"description=List of addresses"`
}

// Instructor handles nested objects automatically
var user User
_, err := client.CreateChatCompletion(ctx, request, &user)
```

### Union types for agent patterns

Build flexible AI agents that can choose between multiple tools or actions:

```go
type SearchTool struct {
    Type  string `json:"type" jsonschema:"const=search"`
    Query string `json:"query" jsonschema:"description=Search query"`
}

type FinishTool struct {
    Type   string `json:"type" jsonschema:"const=finish"`
    Answer string `json:"answer" jsonschema:"description=Final answer"`
}

// Agent loop
for turn := 0; turn < maxTurns; turn++ {
    action, _, err := client.CreateChatCompletionUnion(
        ctx,
        request,
        instructor.UnionOptions{
            Discriminator: "type",
            Variants:      []any{SearchTool{}, FinishTool{}},
        },
    )

    switch tool := action.(type) {
    case SearchTool:
        result := executeSearch(tool.Query)
        conversation.AddUserMessage(result)
    case FinishTool:
        return tool.Answer, nil
    }
}
```

### Multi-provider conversation history

Unified conversation API that works across all providers:

```go
import "github.com/567-labs/instructor-go/pkg/instructor/core"

// Create a conversation with a system prompt
conversation := core.NewConversation("You are a helpful assistant")

// Add messages
conversation.AddUserMessage("What's the weather in SF?")

// Vision support
conversation.AddUserMessageWithImageURLs(
    "What's in this image?",
    "https://example.com/image.jpg",
)

// Convert to provider-specific format
messages := instructor_openai.ConversationToMessages(conversation)
// OR
system, messages := instructor_anthropic.ConversationToMessages(conversation)
// OR
contents := instructor_google.ConversationToContents(conversation)
```

### Token usage tracking

Automatic token counting across retries:

```go
resp, err := client.CreateChatCompletion(ctx, request, &user)

fmt.Printf("Input tokens: %d\n", resp.Usage.PromptTokens)
fmt.Printf("Output tokens: %d\n", resp.Usage.CompletionTokens)
fmt.Printf("Total tokens: %d\n", resp.Usage.TotalTokens)
// Usage is summed across all retry attempts
```

## Used in production

Trusted by developers building production AI applications in Go:

* **Type-safe** by design - catch errors at compile time
* **Zero reflection overhead** - uses code generation where possible
* **Battle-tested** across multiple LLM providers
* **Enterprise-ready** with comprehensive error handling

## Get started

### Basic extraction

Extract structured data from any text:

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/567-labs/instructor-go/pkg/instructor"
    "github.com/sashabaranov/go-openai"
)

type Product struct {
    Name    string  `json:"name"     jsonschema:"description=Product name"`
    Price   float64 `json:"price"    jsonschema:"description=Price in USD"`
    InStock bool    `json:"in_stock" jsonschema:"description=Availability status"`
}

func main() {
    ctx := context.Background()
    client := instructor.FromOpenAI(openai.NewClient(os.Getenv("OPENAI_API_KEY")))

    var product Product
    _, err := client.CreateChatCompletion(
        ctx,
        openai.ChatCompletionRequest{
            Model: openai.GPT4o,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "iPhone 15 Pro, $999, available now",
                },
            },
        },
        &product,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", product)
    // Product{Name: "iPhone 15 Pro", Price: 999.0, InStock: true}
}
```

### Multiple languages

Instructor's simple API is available in many languages:

* **[Go](https://go.useinstructor.com)** - You are here
* [Python](https://python.useinstructor.com) - The original
* [TypeScript](https://js.useinstructor.com) - Full TypeScript support
* [Ruby](https://ruby.useinstructor.com) - Ruby implementation
* [Elixir](https://hex.pm/packages/instructor) - Elixir implementation
* [Rust](https://rust.useinstructor.com) - Rust implementation

### Learn more

* [Documentation](https://pkg.go.dev/github.com/instructor-ai/instructor-go/pkg/instructor) - Full API reference
* [Examples](examples/README.md) - Copy-paste recipes for common patterns
* [Discord](https://discord.gg/UD9GPjbs8c) - Get help from the community

## Why use Instructor Go over alternatives?

**vs Raw JSON mode**: Instructor provides automatic validation, retries, streaming, and nested object support. No manual schema writing or parsing.

**vs LangChain Go (or other frameworks)**: Instructor is focused on one thing - structured extraction. It's lighter, faster, and easier to debug with full type safety.

**vs Custom solutions**: Battle-tested across multiple providers and edge cases. Handles retries, validation, and provider differences automatically.

**vs Python/JS versions**: Native Go performance and type safety. No runtime overhead, compile-time guarantees, and seamless integration with Go codebases.

## Examples

See [`examples/`](examples/) for complete working examples:

* [`examples/user/`](examples/user/) - Basic extraction
* [`examples/agent/`](examples/agent/) - Union types and agent loops
* [`examples/anthropic_agent/`](examples/anthropic_agent/) - Multi-provider agents
* [`examples/streaming/`](examples/streaming/) - Streaming responses
* More examples in the [`examples/`](examples/) directory

## Contributing

We welcome contributions! Check out our [good first issues](https://github.com/567-labs/instructor-go/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) to get started.

## License

MIT License - see [LICENSE](LICENSE) for details.

***

<p align="center">
Built by the Instructor community. Part of the <a href="https://github.com/567-labs">Instructor ecosystem</a>.
</p>
