package google

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	"google.golang.org/genai"
)

func TestCountUsageFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		want     core.UsageSum
	}{
		{"normal", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4, TotalTokenCount: 14}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"authoritative total", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4, ThoughtsTokenCount: 6, TotalTokenCount: 20}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 20, UsageKnown: true}},
		{"total only", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{TotalTokenCount: 14}}, core.UsageSum{TotalTokens: 14, UsageKnown: true}},
		{"components only", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"input only", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10}}, core.UsageSum{InputTokens: 10, TotalTokens: 10}},
		{"output only", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{CandidatesTokenCount: 4}}, core.UsageSum{OutputTokens: 4, TotalTokens: 4}},
		{"empty metadata", &GoogleResponse{UsageMetadata: &genai.GenerateContentResponseUsageMetadata{}}, core.UsageSum{}},
		{"missing metadata", &GoogleResponse{}, core.UsageSum{}},
		{"nil response", nil, core.UsageSum{}},
		{"typed nil response", (*GoogleResponse)(nil), core.UsageSum{}},
		{"wrong response type", struct{}{}, core.UsageSum{}},
	}
	provider := &InstructorGoogle{}
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
		name     string
		metadata *genai.GenerateContentResponseUsageMetadata
		input    int32
		output   int32
		total    int32
	}{
		{"normal", &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4, TotalTokenCount: 14}, 12, 7, 19},
		{"authoritative total", &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4, TotalTokenCount: 20}, 12, 7, 25},
		{"components only", &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 10, CandidatesTokenCount: 4}, 12, 7, 19},
		{"total only", &genai.GenerateContentResponseUsageMetadata{TotalTokenCount: 14}, 2, 3, 19},
		{"empty metadata", &genai.GenerateContentResponseUsageMetadata{}, 2, 3, 5},
		{"missing metadata", nil, 2, 3, 5},
	}
	provider := &InstructorGoogle{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := &genai.Candidate{}
			response := &GoogleResponse{Candidates: []*genai.Candidate{candidate}, UsageMetadata: tt.metadata}
			got, err := provider.AddUsageSumToResponse(response, &core.UsageSum{InputTokens: 2, OutputTokens: 3, TotalTokens: 5})
			if err != nil || got != response {
				t.Fatalf("response = %v, error = %v", got, err)
			}
			metadata := response.UsageMetadata
			if metadata == nil || metadata.PromptTokenCount != tt.input || metadata.CandidatesTokenCount != tt.output || metadata.TotalTokenCount != tt.total {
				t.Fatalf("usage = %+v, want (%v, %v, %v)", metadata, tt.input, tt.output, tt.total)
			}
			if len(response.Candidates) != 1 || response.Candidates[0] != candidate {
				t.Fatal("successful response candidates were changed")
			}
		})
	}
	var response *GoogleResponse
	if got, err := provider.AddUsageSumToResponse(response, &core.UsageSum{}); err != nil || got != response {
		t.Fatalf("nil response = %v, error = %v", got, err)
	}
}
