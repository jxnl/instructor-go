package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

type tokenBudgetStreamInstructor struct {
	tokenBudgetInstructor
	streamCalls int
	streamErr   error
}

func (m *tokenBudgetStreamInstructor) InternalChatStream(context.Context, interface{}, *Schema) (<-chan string, error) {
	m.streamCalls++
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	chunks := make(chan string, 1)
	chunks <- `{"items": [{"type":"person","name":"Alice"}]}`
	close(chunks)
	return chunks, nil
}

func TestTokenBudgetRejectsStreamingBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		budget int
	}{
		{"positive", 100},
		{"zero", 0},
		{"negative", -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &tokenBudgetStreamInstructor{tokenBudgetInstructor: tokenBudgetInstructor{budget: &tc.budget}}
			stream, err := ChatStreamHandler(m, context.Background(), mockRequest{}, tokenBudgetVariant{})
			if err == nil || stream != nil || m.streamCalls != 0 {
				t.Fatalf("stream=%v error=%v provider requests=%d", stream, err, m.streamCalls)
			}
			if got := errors.Is(err, ErrTokenBudgetStreaming); got != (tc.budget > 0) {
				t.Fatalf("streaming budget error=%t for budget %d", got, tc.budget)
			}
			var budgetErr *TokenBudgetError
			if errors.As(err, &budgetErr) {
				t.Fatalf("configuration error must not report usage exhaustion: %v", err)
			}
		})
	}
}

func TestTokenBudgetUnsetPreservesStreaming(t *testing.T) {
	m := &tokenBudgetStreamInstructor{}
	stream, err := ChatStreamHandler(m, context.Background(), mockRequest{}, tokenBudgetVariant{})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case value, ok := <-stream:
		person, valid := value.(*tokenBudgetVariant)
		if !ok || !valid || person.Type != "person" || person.Name != "Alice" {
			t.Fatalf("streamed value=%#v open=%t", value, ok)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not produce its result")
	}
	select {
	case value, ok := <-stream:
		if ok {
			t.Fatalf("unexpected extra streamed result: %#v", value)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not close")
	}
	if m.streamCalls != 1 {
		t.Fatalf("provider requests=%d, want 1", m.streamCalls)
	}
}

func TestTokenBudgetStreamContextPreventsRequests(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "canceled"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			want := context.Canceled
			cancel()
			if deadline {
				ctx, cancel = context.WithDeadline(context.Background(), time.Unix(0, 0))
				defer cancel()
				want = context.DeadlineExceeded
			}
			m := &tokenBudgetStreamInstructor{}
			stream, err := ChatStreamHandler(m, ctx, mockRequest{}, tokenBudgetVariant{})
			if !errors.Is(err, want) || stream != nil || m.streamCalls != 0 {
				t.Fatalf("stream=%v error=%v provider requests=%d", stream, err, m.streamCalls)
			}
		})
	}
}

func TestTokenBudgetStreamPreservesProviderError(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	m := &tokenBudgetStreamInstructor{streamErr: providerErr}
	stream, err := ChatStreamHandler(m, context.Background(), mockRequest{}, tokenBudgetVariant{})
	if !errors.Is(err, providerErr) || stream != nil || m.streamCalls != 1 {
		t.Fatalf("stream=%v error=%v provider requests=%d", stream, err, m.streamCalls)
	}
}
