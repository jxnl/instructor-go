package main

import (
	"context"
	"fmt"
	"os"

	"github.com/567-labs/instructor-go/pkg/instructor"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// Define union variants with discriminators
type SearchTool struct {
	Type  string `json:"type" jsonschema:"const=search,description=Type of tool"`
	Query string `json:"query" jsonschema:"description=Search query to execute"`
}

type LookupTool struct {
	Type    string `json:"type" jsonschema:"const=lookup,description=Type of tool"`
	Keyword string `json:"keyword" jsonschema:"description=Keyword to look up"`
}

type Result struct {
	Type   string `json:"type" jsonschema:"const=result,description=Type of response"`
	Answer string `json:"answer" jsonschema:"description=Final answer"`
}

func main() {
	// This example demonstrates that Anthropic union types now work correctly
	// The bug was: "messages.1.content: Input should be a valid list"
	// This occurred when retry logic tried to append error messages

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY not set - skipping live test")
		fmt.Println("To test with real API, set ANTHROPIC_API_KEY environment variable")
		return
	}

	ctx := context.Background()

	// Create instructor client with Anthropic
	client := instructor.FromAnthropic(
		anthropic.NewClient(apiKey),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3), // Retries now work correctly!
	)

	// Define union variants
	variants := []any{SearchTool{}, LookupTool{}, Result{}}

	// Create request
	request := anthropic.MessagesRequest{
		Model:  anthropic.Model("claude-sonnet-4"),
		System: "You are a helpful assistant that can search, lookup, or provide final answers.",
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage("What is the capital of France?"),
		},
		MaxTokens: 1024,
	}

	// Call with union type extraction
	action, response, err := client.CreateMessagesUnion(
		ctx,
		request,
		instructor.UnionOptions{
			Discriminator: "type",
			Variants:      variants,
		},
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Type switch on the result
	switch tool := action.(type) {
	case SearchTool:
		fmt.Printf("✓ SearchTool selected:\n")
		fmt.Printf("  Query: %s\n", tool.Query)
	case LookupTool:
		fmt.Printf("✓ LookupTool selected:\n")
		fmt.Printf("  Keyword: %s\n", tool.Keyword)
	case Result:
		fmt.Printf("✓ Result returned:\n")
		fmt.Printf("  Answer: %s\n", tool.Answer)
	default:
		fmt.Printf("Unexpected type: %T\n", action)
	}

	fmt.Printf("\nUsage: %d input tokens, %d output tokens\n",
		response.Usage.InputTokens,
		response.Usage.OutputTokens,
	)
}
