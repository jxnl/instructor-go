package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

func (i *InstructorAnthropic) CreateMessages(ctx context.Context, request anthropic.MessagesRequest, responseType any) (response anthropic.MessagesResponse, err error) {

	resp, err := core.ChatHandler(i, ctx, request, responseType)
	if err != nil {
		if resp == nil {
			return anthropic.MessagesResponse{}, err
		}
		return *nilAnthropicRespWithUsage(resp.(*anthropic.MessagesResponse)), err
	}

	response = *(resp.(*anthropic.MessagesResponse))

	return response, nil
}

func (i *InstructorAnthropic) InternalChat(ctx context.Context, request interface{}, schema *core.Schema) (string, interface{}, error) {

	req, ok := request.(anthropic.MessagesRequest)
	if !ok {
		return "", nil, fmt.Errorf("invalid request type for %s client", i.Provider())
	}

	if req.Stream {
		return "", nil, errors.New("streaming is not supported by this method; use CreateChatCompletionStream instead")
	}

	switch i.Mode() {
	case core.ModeToolCall:
		return i.completionToolCall(ctx, &req, schema)
	case core.ModeJSONSchema:
		return i.completionJSONSchema(ctx, &req, schema)
	default:
		return "", nil, fmt.Errorf("mode '%s' is not supported for %s", i.Mode(), i.Provider())
	}
}

func (i *InstructorAnthropic) completionToolCall(ctx context.Context, request *anthropic.MessagesRequest, schema *core.Schema) (string, *anthropic.MessagesResponse, error) {

	request.Tools = []anthropic.ToolDefinition{}

	for _, function := range schema.Functions {
		t := anthropic.ToolDefinition{
			Name:        function.Name,
			Description: function.Description,
			InputSchema: function.Parameters,
		}
		request.Tools = append(request.Tools, t)
	}

	resp, err := i.Client.CreateMessages(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	for _, c := range resp.Content {
		if c.Type != anthropic.MessagesContentTypeToolUse {
			// Skip non tool responses
			continue
		}

		toolInput, err := json.Marshal(c.Input)
		if err != nil {
			return "", nilAnthropicRespWithUsage(&resp), err
		}
		// TODO: handle more than 1 tool use
		return string(toolInput), &resp, nil
	}

	return "", nilAnthropicRespWithUsage(&resp), errors.New("more than 1 tool response at a time is not implemented")

}

func (i *InstructorAnthropic) completionJSONSchema(ctx context.Context, request *anthropic.MessagesRequest, schema *core.Schema) (string, *anthropic.MessagesResponse, error) {

	system := fmt.Sprintf(`
Please responsd with json in the following json_schema:

%s

Make sure to return an instance of the JSON, not the schema itself.
`, schema.String)

	if request.System == "" {
		request.System = system
	} else {
		request.System += system
	}

	resp, err := i.Client.CreateMessages(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	text := resp.Content[0].Text

	return *text, &resp, nil
}

func (i *InstructorAnthropic) EmptyResponseWithUsageSum(usage *core.UsageSum) interface{} {
	return &anthropic.MessagesResponse{
		Usage: anthropic.MessagesUsage{
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
		},
	}
}

func (i *InstructorAnthropic) EmptyResponseWithResponseUsage(response interface{}) interface{} {
	resp, ok := response.(*anthropic.MessagesResponse)
	if !ok || resp == nil {
		return nil
	}

	return &anthropic.MessagesResponse{
		Usage: resp.Usage,
	}
}

func (i *InstructorAnthropic) AddUsageSumToResponse(response interface{}, usage *core.UsageSum) (interface{}, error) {
	resp, ok := response.(*anthropic.MessagesResponse)
	if !ok {
		return response, fmt.Errorf("internal type error: expected *anthropic.MessagesResponse, got %T", response)
	}

	resp.Usage.InputTokens += usage.InputTokens
	resp.Usage.OutputTokens += usage.OutputTokens

	return response, nil
}

func (i *InstructorAnthropic) CountUsageFromResponse(response interface{}, usage *core.UsageSum) *core.UsageSum {
	resp, ok := response.(*anthropic.MessagesResponse)
	if !ok {
		return usage
	}

	usage.InputTokens += resp.Usage.InputTokens
	usage.OutputTokens += resp.Usage.OutputTokens

	return usage
}

func nilAnthropicRespWithUsage(resp *anthropic.MessagesResponse) *anthropic.MessagesResponse {
	if resp == nil {
		return nil
	}

	return &anthropic.MessagesResponse{
		Usage: resp.Usage,
	}
}
