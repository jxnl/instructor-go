package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"github.com/invopop/jsonschema"
	openai "github.com/sashabaranov/go-openai"
)

type ResponseFormatSchemaWrapper struct {
	Type                 string                  `json:"type"`
	Required             []string                `json:"required"`
	AdditionalProperties bool                    `json:"additionalProperties"`
	Properties           *jsonschema.Definitions `json:"properties"`
	Definitions          *jsonschema.Definitions `json:"$defs"`
}

func (i *InstructorOpenAI) CreateChatCompletion(
	ctx context.Context,
	request openai.ChatCompletionRequest,
	responseType any,
) (response openai.ChatCompletionResponse, err error) {

	resp, err := core.ChatHandler(i, ctx, request, responseType)
	if err != nil {
		if resp == nil {
			return openai.ChatCompletionResponse{}, err
		}
		return *nilOpenaiRespWithUsage(resp.(*openai.ChatCompletionResponse)), err
	}

	response = *(resp.(*openai.ChatCompletionResponse))

	return response, nil
}

func (i *InstructorOpenAI) InternalChat(ctx context.Context, request interface{}, schema *core.Schema) (string, interface{}, error) {

	req, ok := request.(openai.ChatCompletionRequest)
	if !ok {
		return "", nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	if req.Stream {
		return "", nil, errors.New("streaming is not supported by this method; use CreateChatCompletionStream instead")
	}

	switch i.Mode() {
	case core.ModeToolCall:
		return i.chatToolCall(ctx, &req, schema, false)
	case core.ModeToolCallStrict:
		return i.chatToolCall(ctx, &req, schema, true)
	case core.ModeJSON:
		return i.chatJSON(ctx, &req, schema, false)
	case core.ModeJSONStrict:
		return i.chatJSON(ctx, &req, schema, true)
	case core.ModeJSONSchema:
		return i.chatJSONSchema(ctx, &req, schema)
	default:
		return "", nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorOpenAI) chatToolCall(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema, strict bool) (string, *openai.ChatCompletionResponse, error) {

	request.Tools = createOpenAITools(schema, strict)

	resp, err := i.Client.CreateChatCompletion(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	var toolCalls []openai.ToolCall
	for _, choice := range resp.Choices {
		toolCalls = choice.Message.ToolCalls

		if len(toolCalls) >= 1 {
			break
		}
	}

	numTools := len(toolCalls)

	if numTools < 1 {
		return "", nilOpenaiRespWithUsage(&resp), errors.New("received no tool calls from model, expected at least 1")
	}

	if numTools == 1 {
		return toolCalls[0].Function.Arguments, &resp, nil
	}

	// numTools >= 1

	jsonArray := make([]map[string]interface{}, len(toolCalls))

	for i, toolCall := range toolCalls {
		var jsonObj map[string]interface{}
		err = json.Unmarshal([]byte(toolCall.Function.Arguments), &jsonObj)
		if err != nil {
			return "", nilOpenaiRespWithUsage(&resp), err
		}
		jsonArray[i] = jsonObj
	}

	resultJSON, err := json.Marshal(jsonArray)
	if err != nil {
		return "", nilOpenaiRespWithUsage(&resp), err
	}

	return string(resultJSON), &resp, nil
}

func (i *InstructorOpenAI) chatJSON(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema, strict bool) (string, *openai.ChatCompletionResponse, error) {

	structName := schema.NameFromRef()

	request.Messages = core.Prepend(request.Messages, *createJSONMessage(schema))

	if strict {
		schemaWrapper := ResponseFormatSchemaWrapper{
			Type:        "object",
			Required:    []string{structName},
			Definitions: &schema.Schema.Definitions,
			Properties: &jsonschema.Definitions{
				structName: schema.Definitions[structName],
			},
			AdditionalProperties: false,
		}

		schemaJSON, _ := json.Marshal(schemaWrapper)
		schemaRaw := json.RawMessage(schemaJSON)

		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:        structName,
				Description: schema.Description,
				Schema:      schemaRaw,
				Strict:      true,
			},
		}
	} else {
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	resp, err := i.Client.CreateChatCompletion(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	text := resp.Choices[0].Message.Content

	if strict {
		resMap := make(map[string]any)
		_ = json.Unmarshal([]byte(text), &resMap)

		cleanedText, _ := json.Marshal(resMap[structName])
		text = string(cleanedText)
	}

	return text, &resp, nil
}

func (i *InstructorOpenAI) chatJSONSchema(ctx context.Context, request *openai.ChatCompletionRequest, schema *core.Schema) (string, *openai.ChatCompletionResponse, error) {

	request.Messages = core.Prepend(request.Messages, *createJSONMessage(schema))

	resp, err := i.Client.CreateChatCompletion(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	text := resp.Choices[0].Message.Content

	return text, &resp, nil
}

func (i *InstructorOpenAI) EmptyResponseWithUsageSum(usage *core.UsageSum) interface{} {
	return &openai.ChatCompletionResponse{
		Usage: openai.Usage{
			PromptTokens:     usage.InputTokens,
			CompletionTokens: usage.OutputTokens,
			TotalTokens:      usage.TotalTokens,
		},
	}
}

func (i *InstructorOpenAI) EmptyResponseWithResponseUsage(response interface{}) interface{} {
	resp, ok := response.(*openai.ChatCompletionResponse)
	if !ok || resp == nil {
		return nil
	}

	return &openai.ChatCompletionResponse{
		Usage: resp.Usage,
	}
}

func (i *InstructorOpenAI) AddUsageSumToResponse(response interface{}, usage *core.UsageSum) (interface{}, error) {
	resp, ok := response.(*openai.ChatCompletionResponse)
	if !ok {
		return response, fmt.Errorf("internal type error: expected *openai.ChatCompletionResponse, got %T", response)
	}

	resp.Usage.PromptTokens += usage.InputTokens
	resp.Usage.CompletionTokens += usage.OutputTokens
	resp.Usage.TotalTokens += usage.TotalTokens

	return response, nil
}

func (i *InstructorOpenAI) CountUsageFromResponse(response interface{}, usage *core.UsageSum) *core.UsageSum {
	resp, ok := response.(*openai.ChatCompletionResponse)
	if !ok {
		return usage
	}

	usage.InputTokens += resp.Usage.PromptTokens
	usage.OutputTokens += resp.Usage.CompletionTokens
	usage.TotalTokens += resp.Usage.TotalTokens

	return usage
}

func createJSONMessage(schema *core.Schema) *openai.ChatCompletionMessage {
	message := fmt.Sprintf(`
Please respond with JSON in the following JSON schema:

%s

Make sure to return an instance of the JSON, not the schema itself
`, schema.String)

	msg := &openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: message,
	}

	return msg
}

func createOpenAITools(schema *core.Schema, strict bool) []openai.Tool {
	tools := make([]openai.Tool, 0, len(schema.Functions))
	for _, function := range schema.Functions {
		f := openai.FunctionDefinition{
			Name:        function.Name,
			Description: function.Description,
			Parameters:  function.Parameters,
			Strict:      strict,
		}
		t := openai.Tool{
			Type:     "function",
			Function: &f,
		}
		tools = append(tools, t)
	}
	return tools
}

func nilOpenaiRespWithUsage(resp *openai.ChatCompletionResponse) *openai.ChatCompletionResponse {
	if resp == nil {
		return nil
	}

	return &openai.ChatCompletionResponse{
		Usage: resp.Usage,
	}
}
