package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

// Error constants for Google streaming
const (
	ErrIteratorDone = "iterator done"
)

func (i *InstructorGoogle) CreateChatCompletionStream(
	ctx context.Context,
	request GoogleRequest,
	responseType any,
) (stream <-chan string, err error) {

	ch, err := core.ChatStreamHandler(i, ctx, request, responseType)
	if err != nil {
		return nil, err
	}

	// Convert interface{} channel to string channel
	stringCh := make(chan string)
	go func() {
		defer close(stringCh)
		for msg := range ch {
			if str, ok := msg.(string); ok {
				stringCh <- str
			}
		}
	}()

	return stringCh, nil
}

func (i *InstructorGoogle) InternalChatStream(ctx context.Context, request interface{}, schema *core.Schema) (<-chan string, error) {

	req, ok := request.(GoogleRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	switch i.Mode() {
	case core.ModeToolCall:
		return i.chatStreamToolCall(ctx, &req, schema, false)
	case core.ModeToolCallStrict:
		return i.chatStreamToolCall(ctx, &req, schema, true)
	case core.ModeJSON:
		return i.chatStreamJSON(ctx, &req, schema, false)
	case core.ModeJSONStrict:
		return i.chatStreamJSON(ctx, &req, schema, true)
	case core.ModeJSONSchema:
		return i.chatStreamJSONSchema(ctx, &req, schema)
	default:
		return nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorGoogle) chatStreamToolCall(ctx context.Context, request *GoogleRequest, schema *core.Schema, strict bool) (<-chan string, error) {
	// Google doesn't support streaming with tool calls in the same way as OpenAI
	// We'll need to implement this differently or return an error
	return nil, errors.New("streaming with tool calls is not supported for Google")
}

func (i *InstructorGoogle) chatStreamJSON(ctx context.Context, request *GoogleRequest, schema *core.Schema, strict bool) (<-chan string, error) {
	structName := schema.NameFromRef()

	// Prepend JSON instruction message
	request.Contents = prependGoogleContents(request.Contents, *createGoogleJSONMessage(schema))

	// Create config (do not use GenerationConfig field)
	config := &genai.GenerateContentConfig{
		SafetySettings: request.SafetySettings,
	}

	// Start streaming
	iter := i.Models.GenerateContentStream(ctx, request.Model, request.Contents, config)

	ch := make(chan string)

	go func() {
		defer close(ch)
		var fullText string

		// Use the iterator with a yield function
		iter(func(resp *genai.GenerateContentResponse, err error) bool {
			if err != nil {
				// Handle end of stream or error
				if err.Error() == ErrIteratorDone {
					// Process the complete response
					if strict {
						resMap := make(map[string]any)
						_ = json.Unmarshal([]byte(fullText), &resMap)
						if cleanedText, err := json.Marshal(resMap[structName]); err == nil {
							fullText = string(cleanedText)
						}
					}
					ch <- fullText
				}
				return false // Stop iteration
			}

			// Extract text from response
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					if part.Text != "" {
						fullText += part.Text
						ch <- part.Text
					}
				}
			}
			return true // Continue iteration
		})
	}()

	return ch, nil
}

func (i *InstructorGoogle) chatStreamJSONSchema(ctx context.Context, request *GoogleRequest, schema *core.Schema) (<-chan string, error) {
	request.Contents = prependGoogleContents(request.Contents, *createGoogleJSONMessage(schema))

	// Create config (do not use GenerationConfig field)
	config := &genai.GenerateContentConfig{
		SafetySettings: request.SafetySettings,
	}

	// Start streaming
	iter := i.Models.GenerateContentStream(ctx, request.Model, request.Contents, config)

	ch := make(chan string)

	go func() {
		defer close(ch)

		// Use the iterator with a yield function
		iter(func(resp *genai.GenerateContentResponse, err error) bool {
			if err != nil {
				// Handle end of stream or error
				if err.Error() == ErrIteratorDone {
					return false // Stop iteration
				}
				return false // Stop iteration on error
			}

			// Extract text from response
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					if part.Text != "" {
						ch <- part.Text
					}
				}
			}
			return true // Continue iteration
		})
	}()

	return ch, nil
}
