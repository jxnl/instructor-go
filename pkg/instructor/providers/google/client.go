package google

import (
	"github.com/jxnl/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

type InstructorGoogle struct {
	*genai.Client

	provider    core.Provider
	mode        core.Mode
	maxRetries  int
	tokenBudget *int
	validate    bool
	logger      core.Logger
}

func FromGoogle(client *genai.Client, opts ...core.Options) *InstructorGoogle {
	options := core.MergeOptions(opts...)

	i := &InstructorGoogle{
		Client: client,

		provider:    core.ProviderGoogle,
		mode:        *options.Mode,
		maxRetries:  *options.MaxRetries,
		tokenBudget: options.TokenBudget,
		validate:    *options.Validate,
		logger:      options.Logger,
	}
	return i
}

func (i *InstructorGoogle) Provider() core.Provider {
	return i.provider
}

func (i *InstructorGoogle) Mode() core.Mode {
	return i.mode
}

func (i *InstructorGoogle) MaxRetries() int {
	return i.maxRetries
}

func (i *InstructorGoogle) TokenBudget() *int {
	return i.tokenBudget
}

func (i *InstructorGoogle) Validate() bool {
	return i.validate
}

func (i *InstructorGoogle) Logger() core.Logger {
	return i.logger
}

// AppendErrorToRequest returns nil to use the default handler
func (i *InstructorGoogle) AppendErrorToRequest(request interface{}, failedResponse string, errorMessage string) interface{} {
	return nil // Use default handler
}

// GoogleRequest represents a request to the Google AI API
type GoogleRequest struct {
	Model            string                  `json:"model"`
	Contents         []*genai.Content        `json:"contents"`
	GenerationConfig *genai.GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []*genai.SafetySetting  `json:"safetySettings,omitempty"`
}

// GoogleResponse represents a response from the Google AI API
type GoogleResponse struct {
	Candidates    []*genai.Candidate                          `json:"candidates"`
	UsageMetadata *genai.GenerateContentResponseUsageMetadata `json:"usageMetadata,omitempty"`
}

func (i *InstructorGoogle) EmptyResponseWithUsageSum(usage *core.UsageSum) interface{} {
	return &GoogleResponse{
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(usage.InputTokens),
			CandidatesTokenCount: int32(usage.OutputTokens),
			TotalTokenCount:      int32(usage.TotalTokens),
		},
	}
}

func (i *InstructorGoogle) EmptyResponseWithResponseUsage(response interface{}) interface{} {
	resp, ok := response.(*GoogleResponse)
	if !ok || resp == nil {
		return nil
	}

	return &GoogleResponse{
		UsageMetadata: resp.UsageMetadata,
	}
}

func (i *InstructorGoogle) AddUsageSumToResponse(response interface{}, usage *core.UsageSum) (interface{}, error) {
	resp, ok := response.(*GoogleResponse)
	if !ok || resp == nil {
		return response, nil
	}

	if usage == nil {
		return response, nil
	}
	if resp.UsageMetadata == nil {
		resp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{}
	}
	if resp.UsageMetadata.TotalTokenCount <= 0 {
		resp.UsageMetadata.TotalTokenCount = resp.UsageMetadata.PromptTokenCount + resp.UsageMetadata.CandidatesTokenCount
	}

	resp.UsageMetadata.PromptTokenCount += int32(usage.InputTokens)
	resp.UsageMetadata.CandidatesTokenCount += int32(usage.OutputTokens)
	resp.UsageMetadata.TotalTokenCount += int32(usage.TotalTokens)

	return resp, nil
}

func (i *InstructorGoogle) CountUsageFromResponse(response interface{}, usage *core.UsageSum) *core.UsageSum {
	if usage == nil {
		return nil
	}
	usage.UsageKnown = false
	resp, ok := response.(*GoogleResponse)
	if !ok || resp == nil || resp.UsageMetadata == nil {
		return usage
	}

	input := int(resp.UsageMetadata.PromptTokenCount)
	output := int(resp.UsageMetadata.CandidatesTokenCount)
	total := int(resp.UsageMetadata.TotalTokenCount)
	usage.UsageKnown = total > 0 || (input > 0 && output > 0)
	if total <= 0 {
		total = input + output
	}
	usage.InputTokens += input
	usage.OutputTokens += output
	usage.TotalTokens += total

	return usage
}

func nilGoogleRespWithUsage(resp *GoogleResponse) *GoogleResponse {
	if resp == nil {
		return &GoogleResponse{}
	}
	return resp
}
