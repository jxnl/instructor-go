/*
 *  Original example in Python: https://github.com/jxnl/instructor/blob/11125a7c831a26e2a4deaef4129f2b4845a7e079/examples/auto-ticketer/run.py
 */

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/567-labs/instructor-go/pkg/instructor"
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	instructor_openai "github.com/567-labs/instructor-go/pkg/instructor/providers/openai"
	"github.com/sashabaranov/go-openai"
)

type PriorityEnum string

const (
	High   PriorityEnum = "High"
	Medium PriorityEnum = "Medium"
	Low    PriorityEnum = "Low"
)

type Subtask struct {
	ID   int    `json:"id"      jsonschema:"title=unique identifier for the subtask,description=Unique identifier for the subtask"`
	Name string `json:"name"    jsonschema:"title=name of the subtask,description=Informative title of the subtask"`
}

type Ticket struct {
	ID           int          `json:"id"                        jsonschema:"title=unique identifier for the ticket,description=Unique identifier for the ticket"`
	Name         string       `json:"name"                      jsonschema:"title=name of the task,description=Title of the task"`
	Description  string       `json:"description"               jsonschema:"title=description of the task,description=Detailed description of the task"`
	Priority     PriorityEnum `json:"priority"                  jsonschema:"title=priority level,description=Priority level"`
	Assignees    []string     `json:"assignees"                 jsonschema:"title=list of users assigned to the task,description=List of users assigned to the task"`
	Subtasks     []Subtask    `json:"subtasks"        jsonschema:"title=list of subtasks associated with the main task,description=List of subtasks associated with the main task"`
	Dependencies []int        `json:"dependencies"    jsonschema:"title=list of ticket IDs that this ticket depends on,description=List of ticket IDs that this ticket depends on"`
}

type ActionItems struct {
	Tickets []Ticket `json:"tickets"`
}

func (ai ActionItems) String() string {
	var sb strings.Builder

	for _, ticket := range ai.Tickets {
		sb.WriteString(fmt.Sprintf("Ticket ID: %d\n", ticket.ID))
		sb.WriteString(fmt.Sprintf("  Name: %s\n", ticket.Name))
		sb.WriteString(fmt.Sprintf("  Description: %s\n", ticket.Description))
		sb.WriteString(fmt.Sprintf("  Priority: %s\n", ticket.Priority))
		sb.WriteString(fmt.Sprintf("  Assignees: %s\n", strings.Join(ticket.Assignees, ", ")))

		if len(ticket.Subtasks) > 0 {
			sb.WriteString("  Subtasks:\n")
			for _, subtask := range ticket.Subtasks {
				sb.WriteString(fmt.Sprintf("    - Subtask ID: %d, Name: %s\n", subtask.ID, subtask.Name))
			}
		}

		if len(ticket.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("  Dependencies: %v\n", ticket.Dependencies))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func main() {
	ctx := context.Background()

	client := instructor.FromOpenAI(
		openai.NewClient(os.Getenv("OPENAI_API_KEY")),
		instructor.WithMode(instructor.ModeJSONStrict),
		instructor.WithMaxRetries(0),
	)

	transcript := `
Alice: Hey team, we have several critical tasks we need to tackle for the upcoming release. First, we need to work on improving the authentication system. It's a top priority.

Bob: Got it, Alice. I can take the lead on the authentication improvements. Are there any specific areas you want me to focus on?

Alice: Good question, Bob. We need both a front-end revamp and back-end optimization. So basically, two sub-tasks.

Carol: I can help with the front-end part of the authentication system.

Bob: Great, Carol. I'll handle the back-end optimization then.

Alice: Perfect. Now, after the authentication system is improved, we have to integrate it with our new billing system. That's a medium priority task.

Carol: Is the new billing system already in place?

Alice: No, it's actually another task. So it's a dependency for the integration task. Bob, can you also handle the billing system?

Bob: Sure, but I'll need to complete the back-end optimization of the authentication system first, so it's dependent on that.

Alice: Understood. Lastly, we also need to update our user documentation to reflect all these changes. It's a low-priority task but still important.

Carol: I can take that on once the front-end changes for the authentication system are done. So, it would be dependent on that.

Alice: Sounds like a plan. Let's get these tasks modeled out and get started.
`

	conversation := core.NewConversation("The following is a transcript of a meeting between a manager and their team. The manager is assigning tasks to their team members and creating action items for them to complete.")
	conversation.AddUserMessage(fmt.Sprintf("Create the action items for the following transcript: %s", transcript))

	var actionItems ActionItems
	_, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       openai.GPT4oMini20240718,
			Temperature: .2,
			Messages:    instructor_openai.ConversationToMessages(conversation),
		},
		&actionItems,
	)
	if err != nil {
		panic(err)
	}

	println(actionItems.String())
	/*
		Ticket ID: 1
		  Name: Improve Authentication System
		  Description: Revamp the front-end and optimize the back-end of the authentication system.
		  Priority: high
		  Assignees: Bob, Carol
		  Subtasks:
		    - Subtask ID: 1, Name: Front-end Revamp
		    - Subtask ID: 2, Name: Back-end Optimization

		Ticket ID: 2
		  Name: Integrate Authentication with New Billing System
		  Description: Integrate the improved authentication system with the new billing system.
		  Priority: medium
		  Assignees: Bob
		  Dependencies: [1]

		Ticket ID: 3
		  Name: Update User Documentation
		  Description: Update the user documentation to reflect changes made to the authentication system.
		  Priority: low
		  Assignees: Carol
		  Dependencies: [1]
	*/
}
