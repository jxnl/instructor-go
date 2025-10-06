package anthropic

import (
	anthropic "github.com/liushuangls/go-anthropic/v2"

	"github.com/567-labs/instructor-go/pkg/instructor/core"
)

type InstructorAnthropic struct {
	*anthropic.Client

	provider   core.Provider
	mode       core.Mode
	maxRetries int
	validate   bool
	logger     core.Logger
}

var _ core.Instructor = &InstructorAnthropic{}

func FromAnthropic(client *anthropic.Client, opts ...core.Options) *InstructorAnthropic {

	options := core.MergeOptions(opts...)

	i := &InstructorAnthropic{
		Client: client,

		provider:   core.ProviderAnthropic,
		mode:       *options.Mode,
		maxRetries: *options.MaxRetries,
		validate:   *options.Validate,
		logger:     options.Logger,
	}
	return i
}

func (i *InstructorAnthropic) Provider() core.Provider {
	return i.provider
}

func (i *InstructorAnthropic) MaxRetries() int {
	return i.maxRetries
}

func (i *InstructorAnthropic) Mode() core.Mode {
	return i.mode
}

func (i *InstructorAnthropic) Validate() bool {
	return i.validate
}

func (i *InstructorAnthropic) Logger() core.Logger {
	return i.logger
}

// AppendErrorToRequest implements Anthropic-specific error appending for []MessageContent
func (i *InstructorAnthropic) AppendErrorToRequest(request interface{}, failedResponse string, errorMessage string) interface{} {
	// Type assert to Anthropic's MessagesRequest
	req, ok := request.(anthropic.MessagesRequest)
	if !ok {
		// Try pointer type
		if reqPtr, ok := request.(*anthropic.MessagesRequest); ok {
			result := i.appendErrorToAnthropicRequest(*reqPtr, failedResponse, errorMessage)
			return &result
		}
		return nil // Fall back to default
	}

	return i.appendErrorToAnthropicRequest(req, failedResponse, errorMessage)
}

// appendErrorToAnthropicRequest handles the Anthropic-specific message structure
func (i *InstructorAnthropic) appendErrorToAnthropicRequest(req anthropic.MessagesRequest, failedResponse string, errorMessage string) anthropic.MessagesRequest {
	// Create assistant message with failed response
	assistantMsg := anthropic.Message{
		Role: anthropic.RoleAssistant,
		Content: []anthropic.MessageContent{
			anthropic.NewTextMessageContent(failedResponse),
		},
	}

	// Create user message with error
	userMsg := anthropic.Message{
		Role: anthropic.RoleUser,
		Content: []anthropic.MessageContent{
			anthropic.NewTextMessageContent(errorMessage),
		},
	}

	// Append messages
	req.Messages = append(req.Messages, assistantMsg, userMsg)

	return req
}
