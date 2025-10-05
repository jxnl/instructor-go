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

// Define different notification types

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

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openaiLib.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	// Example 1: Email notification
	fmt.Println("Example 1: Extracting email notification")
	fmt.Println(strings.Repeat("-", 60))

	conversation1 := core.NewConversation()
	conversation1.AddUserMessage("Send an email to john@example.com with subject 'Meeting Tomorrow' and body 'Don't forget our meeting at 2pm'")

	result1, resp1, err := client.CreateChatCompletionUnion(
		ctx,
		openaiLib.ChatCompletionRequest{
			Model:    openaiLib.GPT4oMini,
			Messages: openai.ConversationToMessages(conversation1),
		},
		instructor.UnionOptions{
			Discriminator: "type",
			Variants:      []any{EmailNotification{}, SMSNotification{}, PushNotification{}},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result type: %T\n", result1)
	if email, ok := result1.(EmailNotification); ok {
		fmt.Printf("Email notification:\n")
		fmt.Printf("  To: %s\n", email.To)
		fmt.Printf("  Subject: %s\n", email.Subject)
		fmt.Printf("  Body: %s\n", email.Body)
	}
	fmt.Printf("Tokens: %d\n", resp1.Usage.TotalTokens)

	// Example 2: SMS notification
	fmt.Println("\nExample 2: Extracting SMS notification")
	fmt.Println(strings.Repeat("-", 60))

	conversation2 := core.NewConversation()
	conversation2.AddUserMessage("Text +1-555-0123 saying 'Your order has been delivered'")

	result2, resp2, err := client.CreateChatCompletionUnion(
		ctx,
		openaiLib.ChatCompletionRequest{
			Model:    openaiLib.GPT4oMini,
			Messages: openai.ConversationToMessages(conversation2),
		},
		instructor.UnionOptions{
			Discriminator: "type",
			Variants:      []any{EmailNotification{}, SMSNotification{}, PushNotification{}},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result type: %T\n", result2)
	if sms, ok := result2.(SMSNotification); ok {
		fmt.Printf("SMS notification:\n")
		fmt.Printf("  Phone: %s\n", sms.Phone)
		fmt.Printf("  Message: %s\n", sms.Message)
	}
	fmt.Printf("Tokens: %d\n", resp2.Usage.TotalTokens)

	// Example 3: Push notification
	fmt.Println("\nExample 3: Extracting push notification")
	fmt.Println(strings.Repeat("-", 60))

	conversation3 := core.NewConversation()
	conversation3.AddUserMessage("Send a push notification with title 'New Message' and body 'You have 3 new messages'")

	result3, resp3, err := client.CreateChatCompletionUnion(
		ctx,
		openaiLib.ChatCompletionRequest{
			Model:    openaiLib.GPT4oMini,
			Messages: openai.ConversationToMessages(conversation3),
		},
		instructor.UnionOptions{
			Discriminator: "type",
			Variants:      []any{EmailNotification{}, SMSNotification{}, PushNotification{}},
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result type: %T\n", result3)
	if push, ok := result3.(PushNotification); ok {
		fmt.Printf("Push notification:\n")
		fmt.Printf("  Title: %s\n", push.Title)
		fmt.Printf("  Body: %s\n", push.Body)
	}
	fmt.Printf("Tokens: %d\n", resp3.Usage.TotalTokens)

	fmt.Println("\nAll examples completed successfully!")
}
