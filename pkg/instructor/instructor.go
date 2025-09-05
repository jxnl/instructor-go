package instructor

import (
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/anthropic"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/cohere"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/google"
	"github.com/instructor-ai/instructor-go/pkg/instructor/providers/openai"

	cohereSDK "github.com/cohere-ai/cohere-go/v2/client"
	anthropicSDK "github.com/liushuangls/go-anthropic/v2"
	openaiSDK "github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

// Re-export core types and interfaces
type (
	Instructor = core.Instructor
	Mode       = core.Mode
	Provider   = core.Provider
	Schema     = core.Schema
	Options    = core.Options
	UsageSum   = core.UsageSum
)

// Re-export provider-specific types for backward compatibility
type (
	InstructorOpenAI    = openai.InstructorOpenAI
	InstructorAnthropic = anthropic.InstructorAnthropic
	InstructorCohere    = cohere.InstructorCohere
	InstructorGoogle    = google.InstructorGoogle
)

// Re-export provider-specific request types
type (
	GoogleRequest  = google.GoogleRequest
	GoogleResponse = google.GoogleResponse
)

// Re-export mode constants
const (
	ModeToolCall       = core.ModeToolCall
	ModeToolCallStrict = core.ModeToolCallStrict
	ModeJSON           = core.ModeJSON
	ModeJSONStrict     = core.ModeJSONStrict
	ModeJSONSchema     = core.ModeJSONSchema
	ModeMarkdownJSON   = core.ModeMarkdownJSON
	ModeDefault        = core.ModeDefault
)

// Re-export provider constants
const (
	ProviderOpenAI    = core.ProviderOpenAI
	ProviderAnthropic = core.ProviderAnthropic
	ProviderCohere    = core.ProviderCohere
	ProviderGoogle    = core.ProviderGoogle
)

// Re-export default constants
const (
	DefaultMaxRetries = core.DefaultMaxRetries
	DefaultValidator  = core.DefaultValidator
)

// Re-export option functions
var (
	WithMode       = core.WithMode
	WithMaxRetries = core.WithMaxRetries
	WithValidation = core.WithValidation
)

// Re-export core functions
var (
	NewSchema    = core.NewSchema
	ChatHandler  = core.ChatHandler
	ExtractJSON  = core.ExtractJSON
	MergeOptions = core.MergeOptions
)

// Re-export generic functions
func ToPtr[T any](val T) *T {
	return core.ToPtr(val)
}

// Re-export provider factory functions
func FromOpenAI(client *openaiSDK.Client, opts ...Options) *openai.InstructorOpenAI {
	return openai.FromOpenAI(client, opts...)
}

func FromAnthropic(client *anthropicSDK.Client, opts ...Options) *anthropic.InstructorAnthropic {
	return anthropic.FromAnthropic(client, opts...)
}

func FromCohere(client *cohereSDK.Client, opts ...Options) *cohere.InstructorCohere {
	return cohere.FromCohere(client, opts...)
}

func FromGoogle(client *genai.Client, opts ...Options) *google.InstructorGoogle {
	return google.FromGoogle(client, opts...)
}
