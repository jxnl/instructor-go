package cohere

import (
	cohere "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
)

type InstructorCohere struct {
	*cohere.Client

	provider   core.Provider
	mode       core.Mode
	maxRetries int
	validate   bool
}

var _ core.Instructor = &InstructorCohere{}

func FromCohere(client *cohere.Client, opts ...core.Options) *InstructorCohere {

	options := core.MergeOptions(opts...)

	i := &InstructorCohere{
		Client: client,

		provider:   core.ProviderCohere,
		mode:       *options.Mode,
		maxRetries: *options.MaxRetries,
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
