package anthropic

import (
	anthropic "github.com/liushuangls/go-anthropic/v2"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
)

type InstructorAnthropic struct {
	*anthropic.Client

	provider   core.Provider
	mode       core.Mode
	maxRetries int
	validate   bool
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
