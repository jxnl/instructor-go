package core

import (
	"context"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

type Instructor interface {
	Provider() Provider
	Mode() Mode
	MaxRetries() int
	Validate() bool

	// Chat / Messages

	InternalChat(
		ctx context.Context,
		request interface{},
		schema *Schema,
	) (string, interface{}, error)

	// Streaming Chat / Messages

	InternalChatStream(
		ctx context.Context,
		request interface{},
		schema *Schema,
	) (<-chan string, error)

	// Usage counting

	EmptyResponseWithUsageSum(usage *UsageSum) interface{}
	EmptyResponseWithResponseUsage(response interface{}) interface{}
	AddUsageSumToResponse(response interface{}, usage *UsageSum) (interface{}, error)
	CountUsageFromResponse(response interface{}, usage *UsageSum) *UsageSum
}
