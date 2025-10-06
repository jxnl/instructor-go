package cohere

import (
	"github.com/567-labs/instructor-go/pkg/instructor/core"
	cohere "github.com/cohere-ai/cohere-go/v2/client"
)

type InstructorCohere struct {
	*cohere.Client

	provider   core.Provider
	mode       core.Mode
	maxRetries int
	validate   bool
	logger     core.Logger
}

var _ core.Instructor = &InstructorCohere{}

func FromCohere(client *cohere.Client, opts ...core.Options) *InstructorCohere {

	options := core.MergeOptions(opts...)

	i := &InstructorCohere{
		Client: client,

		provider:   core.ProviderCohere,
		mode:       *options.Mode,
		maxRetries: *options.MaxRetries,
		logger:     options.Logger,
	}
	return i
}

func (i *InstructorCohere) Provider() core.Provider {
	return i.provider
}

func (i *InstructorCohere) Mode() core.Mode {
	return i.mode
}

func (i *InstructorCohere) MaxRetries() int {
	return i.maxRetries
}
func (i *InstructorCohere) Validate() bool {
	return i.validate
}
func (i *InstructorCohere) Logger() core.Logger {
	return i.logger
}

// AppendErrorToRequest returns nil to use the default handler
func (i *InstructorCohere) AppendErrorToRequest(request interface{}, failedResponse string, errorMessage string) interface{} {
	return nil // Use default handler
}
