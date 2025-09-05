package openai

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

func (i *InstructorOpenAI) CreateChatCompletionStream(
	ctx context.Context,
	request openai.ChatCompletionRequest,
	responseType any,
) (stream <-chan any, err error) {

	stream, err = core.ChatStreamHandler(i, ctx, request, responseType)
	if err != nil {
		return nil, err
	}

	return stream, err
}

func (i *InstructorOpenAI) InternalChatStream(ctx context.Context, request interface{}, schema *core.Schema) (<-chan string, error) {

	req, ok := request.(openai.ChatCompletionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	if !req.Stream {
		return nil, errors.New("streaming is not enabled in request type; use CreateChatCompletion for synchronous completion")
	}

	switch i.Mode() {
	case core.ModeToolCall:
		return i.chatToolCallStream(ctx, &req, schema, false)
	case core.ModeToolCallStrict:
		return i.chatToolCallStream(ctx, &req, schema, true)
	case core.ModeJSON:
		return i.chatJSONStream(ctx, &req, schema)
	case core.ModeJSONSchema:
		return i.chatJSONSchemaStream(ctx, &req, schema)
	default:
		return nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorOpenAI) chatToolCallStream(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema, strict bool) (<-chan string, error) {
	request.Tools = createOpenAITools(schema, strict)
	return i.createStream(ctx, request)
}

func (i *InstructorOpenAI) chatJSONStream(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema) (<-chan string, error) {
	request.Messages = core.Prepend(request.Messages, *createJSONMessageStream(schema))
	// Set JSON mode
	request.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	return i.createStream(ctx, request)
}

func (i *InstructorOpenAI) chatJSONSchemaStream(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema) (<-chan string, error) {
	request.Messages = core.Prepend(request.Messages, *createJSONMessageStream(schema))
	return i.createStream(ctx, request)
}

func createJSONMessageStream(schema *core.Schema) *openai.ChatCompletionMessage {
	message := fmt.Sprintf(`
Please respond with a JSON array where the elements following JSON schema:

%s

Make sure to return an array with the elements an instance of the JSON, not the schema itself.
`, schema.String)

	msg := &openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: message,
	}

	return msg
}

func (i *InstructorOpenAI) createStream(ctx context.Context, request *openai.ChatCompletionRequest) (<-chan string, error) {
	stream, err := i.Client.CreateChatCompletionStream(ctx, *request)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)

	go func() {
		defer stream.Close()
		defer close(ch)
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				return
			}
			text := response.Choices[0].Delta.Content
			ch <- text
		}
	}()
	return ch, nil
}
