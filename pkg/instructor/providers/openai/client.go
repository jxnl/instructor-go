package openai

import (
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

type InstructorOpenAI struct {
	*openai.Client

	provider   core.Provider
	mode       core.Mode
	maxRetries int
	validate   bool
}

var _ core.Instructor = &InstructorOpenAI{}

func FromOpenAI(client *openai.Client, opts ...core.Options) *InstructorOpenAI {

	options := core.MergeOptions(opts...)

	i := &InstructorOpenAI{
		Client: client,

		provider:   core.ProviderOpenAI,
		mode:       *options.Mode,
		maxRetries: *options.MaxRetries,
		validate:   *options.Validate,
	}
	return i
}

func (i *InstructorOpenAI) Provider() core.Provider {
	return i.provider
}
func (i *InstructorOpenAI) Mode() core.Mode {
	return i.mode
}
func (i *InstructorOpenAI) MaxRetries() int {
	return i.maxRetries
}
func (i *InstructorOpenAI) Validate() bool {
	return i.validate
}
