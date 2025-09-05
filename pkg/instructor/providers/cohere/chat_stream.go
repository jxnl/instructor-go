package cohere

import (
	"context"
	"errors"
	"fmt"
	"io"

	cohere "github.com/cohere-ai/cohere-go/v2"
	option "github.com/cohere-ai/cohere-go/v2/option"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
)

// ChatStream provides the public API that matches the original signature
func (i *InstructorCohere) ChatStream(
	ctx context.Context,
	request *cohere.ChatStreamRequest,
	responseType any,
	opts ...option.RequestOption,
) (<-chan any, error) {

	stream, err := core.ChatStreamHandler(i, ctx, request, responseType)
	if err != nil {
		return nil, err
	}

	return stream, err
}

func (i *InstructorCohere) InternalChatStream(ctx context.Context, request interface{}, schema *core.Schema) (<-chan string, error) {

	req, ok := request.(*cohere.ChatStreamRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	switch i.Mode() {
	case core.ModeJSON:
		return i.chatJSONStream(ctx, req, schema)
	default:
		return nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorCohere) chatJSONStream(ctx context.Context, request *cohere.ChatStreamRequest, schema *core.Schema) (<-chan string, error) {
	i.addOrConcatJSONSystemPromptStream(request, schema)
	return i.createStream(ctx, request)
}

func (i *InstructorCohere) addOrConcatJSONSystemPromptStream(request *cohere.ChatStreamRequest, schema *core.Schema) {

	schemaPrompt := fmt.Sprintf("```json!Please respond with JSON in the following JSON schema - make sure to return an instance of the JSON, not the schema itself: %s ", schema.String)

	if request.Preamble == nil {
		request.Preamble = &schemaPrompt
	} else {
		request.Preamble = core.ToPtr(*request.Preamble + "\n" + schemaPrompt)
	}
}

func (i *InstructorCohere) createStream(ctx context.Context, request *cohere.ChatStreamRequest) (<-chan string, error) {
	stream, err := i.Client.ChatStream(ctx, request)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)

	go func() {
		defer stream.Close()
		defer close(ch)
		for {
			message, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				return
			}
			switch message.EventType {
			case "stream-start":
				continue
			case "stream-end":
				return
			case "text-generation":
				ch <- message.TextGeneration.Text
			default:
				panic(errors.New("cohere streaming event type not supported by instructor: " + message.EventType))
			}
		}
	}()
	return ch, nil
}
