package anthropic

import (
	"testing"

	"github.com/jxnl/instructor-go/pkg/instructor/core"
	anthropic "github.com/liushuangls/go-anthropic/v2"
)

func TestCountUsageFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response interface{}
		want     core.UsageSum
	}{
		{"normal", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{InputTokens: 10, OutputTokens: 4}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 14, UsageKnown: true}},
		{"cached input", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{InputTokens: 10, OutputTokens: 4, CacheCreationInputTokens: 3, CacheReadInputTokens: 7}}, core.UsageSum{InputTokens: 10, OutputTokens: 4, TotalTokens: 24, UsageKnown: true}},
		{"all input cached", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{OutputTokens: 4, CacheReadInputTokens: 7}}, core.UsageSum{OutputTokens: 4, TotalTokens: 11, UsageKnown: true}},
		{"input only", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{InputTokens: 10}}, core.UsageSum{InputTokens: 10, TotalTokens: 10}},
		{"output only", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{OutputTokens: 4}}, core.UsageSum{OutputTokens: 4, TotalTokens: 4}},
		{"cache only", &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{CacheCreationInputTokens: 3}}, core.UsageSum{TotalTokens: 3}},
		{"missing", &anthropic.MessagesResponse{}, core.UsageSum{}},
		{"nil response", nil, core.UsageSum{}},
		{"typed nil response", (*anthropic.MessagesResponse)(nil), core.UsageSum{}},
		{"wrong response type", struct{}{}, core.UsageSum{}},
	}
	provider := &InstructorAnthropic{}
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
	provider := &InstructorAnthropic{}
	response := &anthropic.MessagesResponse{Usage: anthropic.MessagesUsage{
		InputTokens: 10, OutputTokens: 4, CacheCreationInputTokens: 3, CacheReadInputTokens: 7,
	}}
	got, err := provider.AddUsageSumToResponse(response, &core.UsageSum{InputTokens: 2, OutputTokens: 3, TotalTokens: 5})
	if err != nil || got != response {
		t.Fatalf("response = %v, error = %v", got, err)
	}
	want := anthropic.MessagesUsage{InputTokens: 12, OutputTokens: 7, CacheCreationInputTokens: 3, CacheReadInputTokens: 7}
	if response.Usage != want {
		t.Fatalf("usage = %+v, want %+v", response.Usage, want)
	}
	if _, err := provider.AddUsageSumToResponse((*anthropic.MessagesResponse)(nil), &core.UsageSum{}); err == nil {
		t.Fatal("expected an error for a nil response")
	}
}
