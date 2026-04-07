package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jxnl/instructor-go/pkg/instructor"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY not set")
		return
	}

	// Super simple! Just pass a string to WithLogging()
	client := instructor.FromAnthropic(
		anthropic.NewClient(apiKey),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
		instructor.WithLogging("debug"), // 🎉 That's it!
	)

	ctx := context.Background()

	fmt.Println("Making request with DEBUG logging enabled...")
	fmt.Println("Watch stderr for log output!")
	fmt.Println()

	person := &Person{}
	_, err := client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model("claude-sonnet-4"),
		Messages:  []anthropic.Message{anthropic.NewUserTextMessage("Extract: John Doe is 30 years old")},
		MaxTokens: 1024,
	}, person)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✓ Success: %+v\n", person)
}
