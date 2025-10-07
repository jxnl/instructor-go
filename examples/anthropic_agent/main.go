package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/567-labs/instructor-go/pkg/instructor"
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	anthropic_provider "github.com/567-labs/instructor-go/pkg/instructor/providers/anthropic"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// SearchTool represents a search tool action
type SearchTool struct {
	Type  string `json:"type" jsonschema:"const=search"`
	Query string `json:"query" jsonschema:"description=Search query to execute"`
}

func (s SearchTool) Execute() string {
	return fmt.Sprintf("Search results for '%s': Found information about Go programming language...", s.Query)
}

// FinishTool represents the final answer
type FinishTool struct {
	Type   string `json:"type" jsonschema:"const=finish"`
	Answer string `json:"answer" jsonschema:"description=Final answer to return"`
}

func (f FinishTool) Execute() string {
	return f.Answer
}

// Agent orchestrates tool selection and execution with proper conversation tracking
type Agent struct {
	client       *instructor.InstructorAnthropic
	conversation *core.Conversation
}

func NewAgent(client *instructor.InstructorAnthropic) *Agent {
	return &Agent{
		client: client,
		conversation: core.NewConversation(`You are a helpful agent that can use tools to answer questions.

Available tools:
- search: Search for information
- finish: Return the final answer

IMPORTANT: You MUST always respond by calling one of these tools. Never respond with plain text.
Use tools step by step to gather information, then call "finish" with your final answer.`),
	}
}

func (a *Agent) Run(ctx context.Context, goal string) (string, error) {
	// Add the user goal to conversation
	a.conversation.AddUserMessage(goal)

	maxTurns := 5
	for turn := 0; turn < maxTurns; turn++ {
		fmt.Printf("\n--- Turn %d ---\n", turn+1)

		// Convert conversation to Anthropic format
		system, messages := anthropic_provider.ConversationToMessages(a.conversation)

		// Let the LLM choose a tool
		actions, resp, err := a.client.CreateMessagesUnion(
			ctx,
			anthropic.MessagesRequest{
				Model:     anthropic.ModelClaude3Haiku20240307,
				System:    system,
				Messages:  messages,
				MaxTokens: 1024,
			},
			instructor.UnionOptions{
				Discriminator:   "type",
				Variants:        []any{SearchTool{}, FinishTool{}},
				RequireToolCall: true, // Prevent plain text responses - model must always call a tool
			},
		)
		if err != nil {
			return "", fmt.Errorf("turn %d error: %w", turn, err)
		}

		// IMPORTANT: Add the LLM response to conversation to preserve tool_use blocks
		// This prevents the agent loop bug where the LLM doesn't recognize it already called a tool
		anthropic_provider.AddResponseToConversation(a.conversation, resp)

		fmt.Printf("Tokens used: %d (input: %d, output: %d)\n",
			resp.Usage.InputTokens+resp.Usage.OutputTokens,
			resp.Usage.InputTokens,
			resp.Usage.OutputTokens,
		)

		// For this simple agent, we expect only one action per turn
		if len(actions) == 0 {
			return "", fmt.Errorf("no action returned from LLM")
		}

		action := actions[0]

		// Execute the tool based on its type
		var result string
		var toolUseID string

		// Extract tool use ID from response
		for _, content := range resp.Content {
			if content.Type == anthropic.MessagesContentTypeToolUse {
				toolUseID = content.ID
				break
			}
		}

		switch tool := action.(type) {
		case SearchTool:
			fmt.Printf("Tool: search(%q)\n", tool.Query)
			result = tool.Execute()
			fmt.Printf("Result: %s\n", result)

		case FinishTool:
			fmt.Printf("Tool: finish\n")
			fmt.Printf("Final Answer: %s\n", tool.Answer)
			return tool.Execute(), nil

		default:
			return "", fmt.Errorf("unknown tool type: %T", action)
		}

		// Add tool result to conversation
		// The tool result must be linked to the tool_use ID from the assistant's response
		a.conversation.AddToolResultMessage(toolUseID, result, false)
	}

	return "", fmt.Errorf("reached maximum turns (%d) without completing task", maxTurns)
}

func main() {
	ctx := context.Background()

	// Initialize the instructor client for Anthropic
	client := instructor.FromAnthropic(
		anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	// Create an agent
	agent := NewAgent(client)

	// Run the agent with a goal
	goal := "Search for information about Go programming language and provide a summary."

	fmt.Printf("Goal: %s\n", goal)
	fmt.Println("=============================================================")

	answer, err := agent.Run(ctx, goal)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=============================================================")
	fmt.Printf("\nFinal Answer:\n%s\n", answer)

	// Demonstrate that conversation history properly preserved tool use
	fmt.Println("\n=============================================================")
	fmt.Println("Conversation History (JSON):")
	messages := agent.conversation.GetMessages()
	for i, msg := range messages {
		msgJSON, _ := json.MarshalIndent(map[string]any{
			"role":          msg.Role,
			"content":       msg.Content,
			"contentBlocks": msg.ContentBlocks,
		}, "", "  ")
		fmt.Printf("\nMessage %d:\n%s\n", i, string(msgJSON))
	}
}
