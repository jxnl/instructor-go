package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/openai"
	openaiLib "github.com/sashabaranov/go-openai"
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
	client       *instructor.InstructorOpenAI
	conversation *core.Conversation
}

func NewAgent(client *instructor.InstructorOpenAI) *Agent {
	return &Agent{
		client: client,
		conversation: core.NewConversation(`You are a helpful agent that can use tools to answer questions.
Available tools:
- search: Search for information
- lookup: Look up detailed information about a specific keyword
- finish: Return the final answer

Use the tools step by step to gather information before finishing with your answer.`),
	}
}

func (a *Agent) Run(ctx context.Context, goal string) (string, error) {
	// Add the user goal to conversation
	a.conversation.AddUserMessage(goal)

	maxTurns := 10
	for turn := 0; turn < maxTurns; turn++ {
		fmt.Printf("\n--- Turn %d ---\n", turn+1)

		// Let the LLM choose a tool
		action, resp, err := a.client.CreateChatCompletionUnion(
			ctx,
			openaiLib.ChatCompletionRequest{
				Model:    openaiLib.GPT5Mini,
				Messages: openai.ConversationToMessages(a.conversation),
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

		// Add assistant's tool choice and result to conversation
		a.conversation.AddAssistantMessage(fmt.Sprintf("Using tool: %s", toolName))
		a.conversation.AddUserMessage(fmt.Sprintf("Tool result: %s", result))
	}

	return "", fmt.Errorf("reached maximum turns (%d) without completing task", maxTurns)
}

func main() {
	ctx := context.Background()

	// Initialize the instructor client
	client := instructor.FromOpenAI(
		openaiLib.NewClient(os.Getenv("OPENAI_API_KEY")),
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
