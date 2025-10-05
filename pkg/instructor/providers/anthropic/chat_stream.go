package anthropic

import (
	"context"

	"github.com/567-labs/instructor-go/pkg/instructor/core"
)

func (i *InstructorAnthropic) InternalChatStream(ctx context.Context, request interface{}, schema *core.Schema) (<-chan string, error) {
	panic("unimplemented")
}
