package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/567-labs/instructor-go/pkg/instructor"
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// Example response type
type Person struct {
	Name string `json:"name" jsonschema:"title=Name,description=The person's name"`
	Age  int    `json:"age" jsonschema:"title=Age,description=The person's age"`
}

func main() {
	// Example 1: JSON logging (for production)
	fmt.Println("=== Example 1: JSON Logging (Production) ===")
	jsonLogging()

	// Example 2: Text logging (for development)
	fmt.Println("\n=== Example 2: Text Logging (Development) ===")
	textLogging()

	// Example 3: Custom log levels
	fmt.Println("\n=== Example 3: Debug Logging ===")
	debugLogging()

	// Example 4: Using existing slog.Logger
	fmt.Println("\n=== Example 4: Using Existing slog.Logger ===")
	customSlogLogger()
}

func jsonLogging() {
	// Create a JSON logger for structured logging (production)
	logger := core.NewLogger(os.Stderr, slog.LevelInfo)

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		core.WithMode(core.ModeToolCall),
		core.WithMaxRetries(3),
		core.WithLogger(logger), // Enable logging
	)

	ctx := context.Background()

	resp, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model("claude-sonnet-4"),
		Messages:  []anthropic.Message{anthropic.NewUserTextMessage("Extract: John Doe is 30 years old")},
		MaxTokens: 1024,
	}, &Person{})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %+v\n", resp)
}

func textLogging() {
	// Create a text logger for human-readable output (development)
	logger := core.NewTextLogger(os.Stderr, slog.LevelInfo)

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		core.WithMode(core.ModeToolCall),
		core.WithMaxRetries(3),
		core.WithLogger(logger),
	)

	ctx := context.Background()

	resp, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model("claude-sonnet-4"),
		Messages:  []anthropic.Message{anthropic.NewUserTextMessage("Extract: Jane Smith is 25 years old")},
		MaxTokens: 1024,
	}, &Person{})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %+v\n", resp)
}

func debugLogging() {
	// Use Debug level to see all details including response previews
	logger := core.NewTextLogger(os.Stderr, slog.LevelDebug)

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		core.WithMode(core.ModeToolCall),
		core.WithMaxRetries(3),
		core.WithLogger(logger),
	)

	ctx := context.Background()

	resp, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model("claude-sonnet-4"),
		Messages:  []anthropic.Message{anthropic.NewUserTextMessage("Extract: Bob Wilson is 35 years old")},
		MaxTokens: 1024,
	}, &Person{})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %+v\n", resp)
}

func customSlogLogger() {
	// Use your existing slog.Logger
	myLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Only warnings and errors
	}))

	logger := core.FromSlog(myLogger)

	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		core.WithMode(core.ModeToolCall),
		core.WithMaxRetries(3),
		core.WithLogger(logger),
	)

	ctx := context.Background()

	resp, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model("claude-sonnet-4"),
		Messages:  []anthropic.Message{anthropic.NewUserTextMessage("Extract: Alice Brown is 28 years old")},
		MaxTokens: 1024,
	}, &Person{})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %+v\n", resp)
}
