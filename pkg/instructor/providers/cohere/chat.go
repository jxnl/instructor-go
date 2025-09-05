package cohere

import (
	"context"
	"fmt"

	cohere "github.com/cohere-ai/cohere-go/v2"
	option "github.com/cohere-ai/cohere-go/v2/option"
	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
)

// Chat provides the public API that matches the original signature
func (i *InstructorCohere) Chat(
	ctx context.Context,
	request *cohere.ChatRequest,
	response any,
	opts ...option.RequestOption,
) (*cohere.NonStreamedChatResponse, error) {

	resp, err := core.ChatHandler(i, ctx, request, response)
	if err != nil {
		if resp == nil {
			return &cohere.NonStreamedChatResponse{}, err
		}
		return nilCohereRespWithUsage(resp.(*cohere.NonStreamedChatResponse)), err
	}

	return resp.(*cohere.NonStreamedChatResponse), nil
}

func (i *InstructorCohere) InternalChat(ctx context.Context, request interface{}, schema *core.Schema) (string, interface{}, error) {

	req, ok := request.(*cohere.ChatRequest)
	if !ok {
		return "", nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	switch i.Mode() {
	case core.ModeToolCall:
		return i.chatToolCall(ctx, req, schema)
	case core.ModeJSON:
		return i.chatJSON(ctx, req, schema)
	default:
		return "", nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorCohere) chatToolCall(ctx context.Context, request *cohere.ChatRequest, schema *core.Schema) (string, *cohere.NonStreamedChatResponse, error) {

	request.Tools = []*cohere.Tool{createCohereTools(schema)}

	resp, err := i.Client.Chat(ctx, request)
	if err != nil {
		return "", nil, err
	}

	_ = resp

	// TODO: implement

	panic("tool call not implemented Cohere")

}

func (i *InstructorCohere) chatJSON(ctx context.Context, request *cohere.ChatRequest, schema *core.Schema) (string, *cohere.NonStreamedChatResponse, error) {

	i.addOrConcatJSONSystemPrompt(request, schema)

	resp, err := i.Client.Chat(ctx, request)
	if err != nil {
		return "", nil, err
	}

	return resp.Text, resp, nil
}

func (i *InstructorCohere) addOrConcatJSONSystemPrompt(request *cohere.ChatRequest, schema *core.Schema) {

	schemaPrompt := fmt.Sprintf("```json!Please respond with JSON in the following JSON schema - make sure to return an instance of the JSON, not the schema itself: %s ", schema.String)

	if request.Preamble == nil {
		request.Preamble = &schemaPrompt
	} else {
		request.Preamble = core.ToPtr(*request.Preamble + "\n" + schemaPrompt)
	}
}

func (i *InstructorCohere) EmptyResponseWithUsageSum(usage *core.UsageSum) interface{} {
	return &cohere.NonStreamedChatResponse{
		Meta: &cohere.ApiMeta{
			Tokens: &cohere.ApiMetaTokens{
				InputTokens:  core.ToPtr(float64(usage.InputTokens)),
				OutputTokens: core.ToPtr(float64(usage.OutputTokens)),
			},
		},
	}
}

func (i *InstructorCohere) EmptyResponseWithResponseUsage(response interface{}) interface{} {
	resp, ok := response.(*cohere.NonStreamedChatResponse)
	if !ok || resp == nil {
		return nil
	}

	return &cohere.NonStreamedChatResponse{
		Meta: resp.Meta,
	}
}

func (i *InstructorCohere) AddUsageSumToResponse(response interface{}, usage *core.UsageSum) (interface{}, error) {
	resp, ok := response.(*cohere.NonStreamedChatResponse)
	if !ok {
		return response, fmt.Errorf("internal type error: expected *cohere.NonStreamedChatResponse, got %T", response)
	}

	*resp.Meta.Tokens.InputTokens += float64(usage.InputTokens)
	*resp.Meta.Tokens.OutputTokens += float64(usage.OutputTokens)

	return response, nil
}

func (i *InstructorCohere) CountUsageFromResponse(response interface{}, usage *core.UsageSum) *core.UsageSum {
	resp, ok := response.(*cohere.NonStreamedChatResponse)
	if !ok {
		return usage
	}

	usage.InputTokens += int(*resp.Meta.Tokens.InputTokens)
	usage.OutputTokens += int(*resp.Meta.Tokens.OutputTokens)

	return usage
}

func createCohereTools(schema *core.Schema) *cohere.Tool {

	tool := &cohere.Tool{
		Name:                 "functions",
		Description:          schema.Schema.Description,
		ParameterDefinitions: make(map[string]*cohere.ToolParameterDefinitionsValue),
	}

	for _, function := range schema.Functions {
		parameterDefinition := &cohere.ToolParameterDefinitionsValue{
			Description: core.ToPtr(function.Description),
			Type:        function.Parameters.Type,
		}
		tool.ParameterDefinitions[function.Name] = parameterDefinition
	}

	return tool
}

func nilCohereRespWithUsage(resp *cohere.NonStreamedChatResponse) *cohere.NonStreamedChatResponse {
	if resp == nil {
		return nil
	}

	return &cohere.NonStreamedChatResponse{
		Meta: resp.Meta,
	}
}
