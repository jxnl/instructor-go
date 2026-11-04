# instructor-go

_Structured large language model (LLM) extraction in Go, designed for simplicity, transparency, and control._

---

[![Twitter Follow](https://img.shields.io/twitter/follow/jxnlco?style=social)](https://twitter.com/jxnlco)
[![LinkedIn Follow](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/robby-horvath/)
[![Documentation](https://img.shields.io/badge/docs-available-brightgreen)](https://go.useinstructor.com)
[![GitHub issues](https://img.shields.io/github/issues/jxnl/instructor-go.svg)](https://github.com/jxnl/instructor-go/issues)
[![Discord](https://img.shields.io/discord/1192334452110659664?label=discord)](https://discord.gg/UD9GPjbs8c)

Check us out in [Python](https://python.useinstructor.com/), [JS/TS](https://js.useinstructor.com/), [Elixir](https://github.com/thmsmlr/instructor_ex/) and [PHP](https://github.com/cognesy/instructor-php/).

If you want to port Instructor to another language, please reach out to us on [Twitter](https://twitter.com/jxnlco) we'd love to help you get started!

## Usage

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jxnl/instructor-go/pkg/instructor"
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

## Token budgets for retries

Add `instructor.WithTokenBudget(2000)` alongside `WithMaxRetries` to guard
non-streaming JSON-decoding and validation retries, including union extraction.
The budget resets for each call. The first request runs normally, and successful
results are returned even if they exceed the budget. Zero and negative budgets
are rejected before a request is made; omitting the option leaves retries unchanged.

After a failed attempt, retries stop when cumulative reported tokens reach the
budget or usage is missing/incomplete. Provider totals are used when available;
otherwise complete input/output counts are summed without double counting.
For non-pointer SDK counts, zero may be indistinguishable from missing usage,
so budgeting conservatively requires a positive total or both counts positive.

Use `errors.As(err, &budgetErr)` with a `*instructor.TokenBudgetError` to inspect
`Budget`, `Usage`, and `UsageKnown`. The error unwraps the last decoding or
validation error. Retry limits and context cancellation/deadlines still apply
independently; an exhausted retry limit retains its existing error.


## Contributing

If you want to help out, checkout some of the issues marked as `good-first-issue` or `help-wanted`. Found [here](https://github.com/jxnl/instructor-go/labels/good%20first%20issue). They could be anything from code improvements, a guest blog post, or a new cook book.

## License

This project is licensed under the terms of the MIT License.
