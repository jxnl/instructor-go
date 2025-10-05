package main

import (
	"context"
	"fmt"
	"os"

	"github.com/567-labs/instructor-go/pkg/instructor"
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	instructor_openai "github.com/567-labs/instructor-go/pkg/instructor/providers/openai"
	"github.com/sashabaranov/go-openai"
)

type SearchType string

const (
	Web   SearchType = "web"
	Image SearchType = "image"
	Video SearchType = "video"
)

type Search struct {
	Topic string     `json:"topic" jsonschema:"title=Topic,description=Topic of the search,example=golang"`
	Query string     `json:"query" jsonschema:"title=Query,description=Query to search for relevant content,example=what is golang"`
	Type  SearchType `json:"type"  jsonschema:"title=Type,description=Type of search,default=web,enum=web,enum=image,enum=video"`
}

func (s *Search) execute() {
	fmt.Printf("Searching for `%s` with query `%s` using `%s`\n", s.Topic, s.Query, s.Type)
}

type Searches = []Search

func segment(ctx context.Context, data string) *Searches {

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeToolCall),
		instructor.WithMaxRetries(3),
	)

	conversation := core.NewConversation()
	conversation.AddUserMessage(fmt.Sprintf("Consider the data below: '\n%s' and segment it into multiple search queries", data))

	var searches Searches
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT4o,
		Messages: instructor_openai.ConversationToMessages(conversation),
	},
		&searches,
	)
	if err != nil {
		panic(err)
	}

	return &searches
}

func main() {
	ctx := context.Background()

	q := "Search for a picture of a cat, a video of a dog, and the taxonomy of each"
	for _, search := range *segment(ctx, q) {
		search.execute()
	}
	/*
		Searching for `cat` with query `picture of a cat` using `image`
		Searching for `dog` with query `video of a dog` using `video`
		Searching for `cat` with query `taxonomy of a cat` using `web`
		Searching for `dog` with query `taxonomy of a dog` using `web`
	*/
}
