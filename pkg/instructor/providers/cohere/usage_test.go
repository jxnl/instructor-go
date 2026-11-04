package cohere

import (
	"testing"

	cohere "github.com/cohere-ai/cohere-go/v2"
	"github.com/jxnl/instructor-go/pkg/instructor/core"
)

func TestCountUsageFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		want     core.UsageSum
	}{
		{"normal", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(10.0), OutputTokens: core.ToPtr(4.0)}}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"explicit zero", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(0.0), OutputTokens: core.ToPtr(0.0)}}}, core.UsageSum{UsageKnown: true}},
		{"zero output", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(10.0), OutputTokens: core.ToPtr(0.0)}}}, core.UsageSum{InputTokens: 10, TotalTokens: 10, UsageKnown: true}},
		{"input only", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(10.0)}}}, core.UsageSum{InputTokens: 10, TotalTokens: 10}},
		{"output only", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{OutputTokens: core.ToPtr(4.0)}}}, core.UsageSum{OutputTokens: 4, TotalTokens: 4}},
		{"missing components", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{}}}, core.UsageSum{}},
		{"missing tokens", &cohere.NonStreamedChatResponse{Meta: &cohere.ApiMeta{}}, core.UsageSum{}},
		{"missing meta", &cohere.NonStreamedChatResponse{}, core.UsageSum{}},
		{"nil response", nil, core.UsageSum{}},
		{"typed nil response", (*cohere.NonStreamedChatResponse)(nil), core.UsageSum{}},
		{"wrong response type", struct{}{}, core.UsageSum{}},
	}
	provider := &InstructorCohere{}
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
		name   string
		meta   *cohere.ApiMeta
		input  float64
		output float64
	}{
		{"normal", &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(10.0), OutputTokens: core.ToPtr(4.0)}}, 12, 7},
		{"input only", &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{InputTokens: core.ToPtr(10.0)}}, 12, 3},
		{"output only", &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{OutputTokens: core.ToPtr(4.0)}}, 2, 7},
		{"missing components", &cohere.ApiMeta{Tokens: &cohere.ApiMetaTokens{}}, 2, 3},
		{"missing tokens", &cohere.ApiMeta{}, 2, 3},
		{"missing meta", nil, 2, 3},
	}
	provider := &InstructorCohere{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &cohere.NonStreamedChatResponse{Meta: tt.meta}
			got, err := provider.AddUsageSumToResponse(response, &core.UsageSum{InputTokens: 2, OutputTokens: 3, TotalTokens: 5})
			if err != nil || got != response {
				t.Fatalf("response = %v, error = %v", got, err)
			}
			if response.Meta == nil || response.Meta.Tokens == nil || response.Meta.Tokens.InputTokens == nil || response.Meta.Tokens.OutputTokens == nil {
				t.Fatalf("usage metadata is incomplete: %+v", response.Meta)
			}
			if *response.Meta.Tokens.InputTokens != tt.input || *response.Meta.Tokens.OutputTokens != tt.output {
				t.Fatalf("usage = (%v, %v), want (%v, %v)", *response.Meta.Tokens.InputTokens, *response.Meta.Tokens.OutputTokens, tt.input, tt.output)
			}
		})
	}
	if _, err := provider.AddUsageSumToResponse((*cohere.NonStreamedChatResponse)(nil), &core.UsageSum{}); err == nil {
		t.Fatal("expected an error for a nil response")
	}
}
