package openai

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	openai "github.com/sashabaranov/go-openai"
)

func TestCountUsageFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		want     core.UsageSum
	}{
		{"normal", &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"authoritative total", &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 20}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 20, UsageKnown: true}},
		{"total only", &openai.ChatCompletionResponse{Usage: openai.Usage{TotalTokens: 14}}, core.UsageSum{TotalTokens: 14, UsageKnown: true}},
		{"components only", &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 10, CompletionTokens: 4}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"input only", &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 10}}, core.UsageSum{InputTokens: 10, TotalTokens: 10}},
		{"output only", &openai.ChatCompletionResponse{Usage: openai.Usage{CompletionTokens: 4}}, core.UsageSum{OutputTokens: 4, TotalTokens: 4}},
		{"missing", &openai.ChatCompletionResponse{}, core.UsageSum{}},
		{"nil response", nil, core.UsageSum{}},
		{"typed nil response", (*openai.ChatCompletionResponse)(nil), core.UsageSum{}},
		{"wrong response type", struct{}{}, core.UsageSum{}},
	}
	provider := &InstructorOpenAI{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &core.UsageSum{InputTokens: 2, OutputTokens: 3, TotalTokens: 5, UsageKnown: true}
			got := provider.CountUsageFromResponse(tt.response, usage)
			want := tt.want
			want.InputTokens += 2
			want.OutputTokens += 3
			want.TotalTokens += 5
			if got != usage || *got != want {
				t.Fatalf("usage = %+v, want %+v in original accumulator", got, want)
			}
		})
	}
}

func TestAddUsageSumToResponse(t *testing.T) {
	tests := []struct {
		name  string
		usage openai.Usage
		want  openai.Usage
	}{
		{"normal", openai.Usage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14}, openai.Usage{PromptTokens: 12, CompletionTokens: 7, TotalTokens: 19}},
		{"components only", openai.Usage{PromptTokens: 10, CompletionTokens: 4}, openai.Usage{PromptTokens: 12, CompletionTokens: 7, TotalTokens: 19}},
		{"total only", openai.Usage{TotalTokens: 14}, openai.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 19}},
		{"missing", openai.Usage{}, openai.Usage{PromptTokens: 2, CompletionTokens: 3, TotalTokens: 5}},
	}
	provider := &InstructorOpenAI{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &openai.ChatCompletionResponse{Usage: tt.usage}
			got, err := provider.AddUsageSumToResponse(response, &core.UsageSum{InputTokens: 2, OutputTokens: 3, TotalTokens: 5})
			if err != nil || got != response {
				t.Fatalf("response = %v, error = %v", got, err)
			}
			if response.Usage != tt.want {
				t.Fatalf("usage = %+v, want %+v", response.Usage, tt.want)
			}
		})
	}
	if _, err := provider.AddUsageSumToResponse((*openai.ChatCompletionResponse)(nil), &core.UsageSum{}); err == nil {
		t.Fatal("expected an error for a nil response")
	}
}
