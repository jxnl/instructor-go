package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	openai "github.com/sashabaranov/go-openai"
)

// Define the tool types with discriminator fields

type SearchTool struct {
	Type  string `json:"type" jsonschema:"const=search"`
	Query string `json:"query" jsonschema:"description=Search query to execute"`
}

func (s SearchTool) Execute() string {
	// Simulate search operation
	return fmt.Sprintf("Search results for: %s\n- Result 1: Go is a programming language\n- Result 2: Go has excellent concurrency support", s.Query)
}

type LookupTool struct {
	Type    string `json:"type" jsonschema:"const=lookup"`
	Keyword string `json:"keyword" jsonschema:"description=Keyword to look up details for"`
}

func (l LookupTool) Execute() string {
	// Simulate lookup operation
	return fmt.Sprintf("Details for '%s':\nGo's interface system allows for flexible and composable designs. Interfaces define behavior contracts that types can implement.", l.Keyword)
}

type FinishTool struct {
	Type   string `json:"type" jsonschema:"const=finish"`
	Answer string `json:"answer" jsonschema:"description=Final answer to return to the user"`
}

func (f FinishTool) Execute() string {
	return f.Answer
}

// Agent orchestrates tool selection and execution
type Agent struct {
	client  *instructor.InstructorOpenAI
	history []openai.ChatCompletionMessage
}

func NewAgent(client *instructor.InstructorOpenAI) *Agent {
	return &Agent{
		client:  client,
		history: []openai.ChatCompletionMessage{},
	}
}

func (a *Agent) Run(ctx context.Context, goal string) (string, error) {
	// Add the user goal to history
	a.history = append(a.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: goal,
	})

	// Add system message for agent behavior
	systemMessage := openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleSystem,
		Content: `You are a helpful agent that can use tools to answer questions.
Available tools:
- search: Search for information
- lookup: Look up detailed information about a specific keyword
- finish: Return the final answer

Use the tools step by step to gather information before finishing with your answer.`,
	}
	messages := append([]openai.ChatCompletionMessage{systemMessage}, a.history...)

	maxTurns := 10
	for turn := 0; turn < maxTurns; turn++ {
		fmt.Printf("\n--- Turn %d ---\n", turn+1)

		// Let the LLM choose a tool
		action, resp, err := a.client.CreateChatCompletionUnion(
			ctx,
			openai.ChatCompletionRequest{
				Model:    openai.GPT5Mini,
				Messages: messages,
			},
			instructor.UnionOptions{
				Discriminator: "type",
				Variants:      []any{SearchTool{}, LookupTool{}, FinishTool{}},
			},
		)
		if err != nil {
			return "", fmt.Errorf("turn %d error: %w", turn, err)
		}

		// Log token usage
		fmt.Printf("Tokens used: %d (input: %d, output: %d)\n",
			resp.Usage.TotalTokens,
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
		)

		// Execute the tool based on its type
		var result string
		var toolName string

		switch tool := action.(type) {
		case SearchTool:
			toolName = "search"
			fmt.Printf("Tool: search(%q)\n", tool.Query)
			result = tool.Execute()
			fmt.Printf("Result: %s\n", result)

		case LookupTool:
			toolName = "lookup"
			fmt.Printf("Tool: lookup(%q)\n", tool.Keyword)
			result = tool.Execute()
			fmt.Printf("Result: %s\n", result)

		case FinishTool:
			fmt.Printf("Tool: finish\n")
			fmt.Printf("Final Answer: %s\n", tool.Answer)
			return tool.Execute(), nil

		default:
			return "", fmt.Errorf("unknown tool type: %T", action)
		}

		// Add assistant's tool choice to history
		a.history = append(a.history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fmt.Sprintf("Using tool: %s", toolName),
		})

		// Add tool result to history
		a.history = append(a.history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Tool result: %s", result),
		})

		// Update messages for next turn
		messages = append([]openai.ChatCompletionMessage{systemMessage}, a.history...)
	}

	return "", fmt.Errorf("reached maximum turns (%d) without completing task", maxTurns)
}

func main() {
	ctx := context.Background()

	// Initialize the instructor client
	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	// Create an agent
	agent := NewAgent(client)

	// Run the agent with a goal
	goal := "Search for information about Go programming language, then lookup details about interfaces, and provide a summary."

	fmt.Printf("Goal: %s\n", goal)
	fmt.Println(strings.Repeat("=", 80))

	answer, err := agent.Run(ctx, goal)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nFinal Answer:\n%s\n", answer)
}
