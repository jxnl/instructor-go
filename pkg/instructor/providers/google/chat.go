package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instructor-ai/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

func (i *InstructorGoogle) CreateChatCompletion(
	ctx context.Context,
	request GoogleRequest,
	responseType any,
) (response GoogleResponse, err error) {
	resp, err := core.ChatHandler(i, ctx, request, responseType)
	if err != nil {
		if resp == nil {
			return GoogleResponse{}, err
		}
		return *nilGoogleRespWithUsage(resp.(*GoogleResponse)), err
	}
	response = *(resp.(*GoogleResponse))
	return response, nil
}

func (i *InstructorGoogle) InternalChat(ctx context.Context, request interface{}, schema *core.Schema) (string, interface{}, error) {
	req, ok := request.(GoogleRequest)
	if !ok {
		return "", nil, fmt.Errorf("invalid request type for %s client", i.Provider())
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

func (i *InstructorGoogle) chatToolCall(ctx context.Context, request *GoogleRequest, schema *core.Schema, strict bool) (string, *GoogleResponse, error) {
	tools := createGoogleTools(schema, strict)
	if request.GenerationConfig == nil {
		request.GenerationConfig = &genai.GenerationConfig{}
	}
	resp, err := i.Models.GenerateContent(ctx, request.Model, request.Contents, &genai.GenerateContentConfig{
		Tools:          tools,
		SafetySettings: request.SafetySettings,
	})
	if err != nil {
		return "", nil, err
	}
	googleResp := &GoogleResponse{
		Candidates:    resp.Candidates,
		UsageMetadata: resp.UsageMetadata,
	}
	var toolCalls []*genai.FunctionCall
	for _, candidate := range resp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					toolCalls = append(toolCalls, part.FunctionCall)
				}
			}
		}
	}
	numTools := len(toolCalls)
	if numTools < 1 {
		return "", nilGoogleRespWithUsage(googleResp), errors.New("received no tool calls from model, expected at least 1")
	}
	if numTools == 1 {
		argsJSON, err := json.Marshal(toolCalls[0].Args)
		if err != nil {
			return "", nilGoogleRespWithUsage(googleResp), err
		}
		return string(argsJSON), googleResp, nil
	}
	jsonArray := make([]map[string]interface{}, len(toolCalls))
	for i, toolCall := range toolCalls {
		argsJSON, err := json.Marshal(toolCall.Args)
		if err != nil {
			return "", nilGoogleRespWithUsage(googleResp), err
		}
		var jsonObj map[string]interface{}
		err = json.Unmarshal(argsJSON, &jsonObj)
		if err != nil {
			return "", nilGoogleRespWithUsage(googleResp), err
		}
		jsonArray[i] = jsonObj
	}
	resultJSON, err := json.Marshal(jsonArray)
	if err != nil {
		return "", nilGoogleRespWithUsage(googleResp), err
	}
	return string(resultJSON), googleResp, nil
}

func (i *InstructorGoogle) chatJSON(ctx context.Context, request *GoogleRequest, schema *core.Schema, strict bool) (string, *GoogleResponse, error) {
	structName := schema.NameFromRef()
	request.Contents = prependGoogleContents(request.Contents, *createGoogleJSONMessage(schema))
	if strict {
		if request.GenerationConfig == nil {
			request.GenerationConfig = &genai.GenerationConfig{}
		}
	}
	resp, err := i.Models.GenerateContent(ctx, request.Model, request.Contents, &genai.GenerateContentConfig{
		SafetySettings: request.SafetySettings,
	})
	if err != nil {
		return "", nil, err
	}
	googleResp := &GoogleResponse{
		Candidates:    resp.Candidates,
		UsageMetadata: resp.UsageMetadata,
	}
	text := ""
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				text = part.Text
				break
			}
		}
	}
	if strict {
		resMap := make(map[string]any)
		_ = json.Unmarshal([]byte(text), &resMap)
		cleanedText, _ := json.Marshal(resMap[structName])
		text = string(cleanedText)
	}
	return text, googleResp, nil
}

func (i *InstructorGoogle) chatJSONSchema(ctx context.Context, request *GoogleRequest, schema *core.Schema) (string, *GoogleResponse, error) {
	request.Contents = prependGoogleContents(request.Contents, *createGoogleJSONMessage(schema))
	resp, err := i.Models.GenerateContent(ctx, request.Model, request.Contents, &genai.GenerateContentConfig{
		SafetySettings: request.SafetySettings,
	})
	if err != nil {
		return "", nil, err
	}
	googleResp := &GoogleResponse{
		Candidates:    resp.Candidates,
		UsageMetadata: resp.UsageMetadata,
	}
	text := ""
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				text = part.Text
				break
			}
		}
	}
	return text, googleResp, nil
}

func createGoogleJSONMessage(schema *core.Schema) *genai.Content {
	schemaJSON, _ := json.Marshal(schema.Schema)
	return &genai.Content{
		Parts: []*genai.Part{
			{
				Text: fmt.Sprintf("You are a helpful assistant that responds with valid JSON according to the following schema:\n\n%s\n\nRespond with valid JSON only.", string(schemaJSON)),
			},
		},
		Role: "user",
	}
}

func createGoogleTools(schema *core.Schema, strict bool) []*genai.Tool {
	// TODO: Convert schema.Schema.Properties to map[string]*genai.Schema if needed
	tool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        schema.NameFromRef(),
				Description: schema.Description,
				Parameters: &genai.Schema{
					Type:       "object",
					Properties: map[string]*genai.Schema{}, // TODO: convert from schema.Schema.Properties
					Required:   []string{},                 // TODO: convert from schema.Schema.Required
				},
			},
		},
	}
	return []*genai.Tool{tool}
}

func prependGoogleContents(contents []*genai.Content, content genai.Content) []*genai.Content {
	return append([]*genai.Content{&content}, contents...)
}
