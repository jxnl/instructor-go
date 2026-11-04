package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

type tokenBudgetAttempt struct {
	text  string
	usage *UsageSum
}

type tokenBudgetInstructor struct {
	mockInstructor
	budget   *int
	attempts []tokenBudgetAttempt
	known    map[*mockResponse]bool
	after    func()
}

func (m *tokenBudgetInstructor) TokenBudget() *int { return m.budget }

func (m *tokenBudgetInstructor) InternalChat(ctx context.Context, request interface{}, schema *Schema) (string, interface{}, error) {
	m.requestHistory = append(m.requestHistory, request)
	if m.responseIndex >= len(m.attempts) {
		return "", nil, errors.New("unexpected extra token budget request")
	}
	attempt := m.attempts[m.responseIndex]
	m.responseIndex++
	var response *mockResponse
	if attempt.usage != nil {
		response = &mockResponse{
			InputTokens:  attempt.usage.InputTokens,
			OutputTokens: attempt.usage.OutputTokens,
			TotalTokens:  attempt.usage.TotalTokens,
		}
		if m.known == nil {
			m.known = make(map[*mockResponse]bool)
		}
		m.known[response] = attempt.usage.UsageKnown
	}
	if m.after != nil {
		m.after()
	}
	if response == nil {
		return attempt.text, nil, nil
	}
	return attempt.text, response, nil
}

func (m *tokenBudgetInstructor) CountUsageFromResponse(response interface{}, usage *UsageSum) *UsageSum {
	usage.UsageKnown = false
	if response, ok := response.(*mockResponse); ok && response != nil {
		usage.InputTokens += response.InputTokens
		usage.OutputTokens += response.OutputTokens
		usage.TotalTokens += response.TotalTokens
		usage.UsageKnown = m.known[response]
	}
	return usage
}

type tokenBudgetVariant struct {
	Type string `json:"type" jsonschema:"const=person"`
	Name string `json:"name" validate:"required"`
}

type tokenBudgetHandler struct {
	name string
	call func(Instructor, context.Context) (interface{}, error)
}

func tokenBudgetHandlers() []tokenBudgetHandler {
	return []tokenBudgetHandler{
		{
			name: "regular",
			call: func(i Instructor, ctx context.Context) (interface{}, error) {
				var person tokenBudgetVariant
				response, err := ChatHandler(i, ctx, mockRequest{}, &person)
				if err == nil && person.Name != "Alice" {
					return response, errors.New("successful response did not decode Alice")
				}
				return response, err
			},
		},
		{
			name: "union",
			call: func(i Instructor, ctx context.Context) (interface{}, error) {
				result, response, err := ChatHandlerUnion(i, ctx, mockRequest{}, UnionOptions{
					Discriminator: "type",
					Variants:      []any{tokenBudgetVariant{}},
				})
				if err == nil {
					if len(result) != 1 {
						return response, errors.New("successful union did not return one variant")
					}
					person, ok := result[0].(tokenBudgetVariant)
					if !ok || person.Name != "Alice" {
						return response, errors.New("successful union did not decode Alice")
					}
				}
				return response, err
			},
		},
	}
}

func newTokenBudgetInstructor(budget *int, retries int, attempts ...tokenBudgetAttempt) *tokenBudgetInstructor {
	return &tokenBudgetInstructor{
		mockInstructor: mockInstructor{maxRetries: retries, mode: ModeJSON, provider: ProviderOpenAI},
		budget:         budget,
		attempts:       attempts,
	}
}

func tokenBudgetUsage(input, output int) *UsageSum {
	return &UsageSum{InputTokens: input, OutputTokens: output, TotalTokens: input + output, UsageKnown: true}
}

func requireTokenBudgetCalls(t *testing.T, m *tokenBudgetInstructor, want int) {
	t.Helper()
	if got := len(m.requestHistory); got != want {
		t.Fatalf("requests = %d, want %d", got, want)
	}
}

func requireTokenBudgetResponseUsage(t *testing.T, response interface{}, want UsageSum) {
	t.Helper()
	got, ok := response.(*mockResponse)
	if !ok || got == nil {
		t.Fatalf("response = %T, want non-nil *mockResponse", response)
	}
	if got.InputTokens != want.InputTokens || got.OutputTokens != want.OutputTokens || got.TotalTokens != want.TotalTokens {
		t.Errorf("response usage = %+v, want input=%d output=%d total=%d", got, want.InputTokens, want.OutputTokens, want.TotalTokens)
	}
}

func requireTokenBudgetError(t *testing.T, err error, budget int, want UsageSum) *TokenBudgetError {
	t.Helper()
	var budgetErr *TokenBudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("error = %v (%T), want *TokenBudgetError", err, err)
	}
	if budgetErr.Budget != budget || budgetErr.Usage != want || budgetErr.UsageKnown != want.UsageKnown {
		t.Errorf("budget error = %+v, want budget=%d usage=%+v known=%v", budgetErr, budget, want, want.UsageKnown)
	}
	return budgetErr
}

func TestTokenBudgetUnset(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		t.Run(handler.name, func(t *testing.T) {
			m := newTokenBudgetInstructor(nil, 2,
				tokenBudgetAttempt{text: `{"name":}`},
				tokenBudgetAttempt{text: `{"name":}`},
				tokenBudgetAttempt{text: `{"type":"person","name":"Alice"}`, usage: tokenBudgetUsage(3, 2)},
			)
			response, err := handler.call(m, context.Background())
			if err != nil {
				t.Fatalf("unset budget must retain retries despite missing usage: %v", err)
			}
			requireTokenBudgetCalls(t, m, 3)
			requireTokenBudgetResponseUsage(t, response, *tokenBudgetUsage(3, 2))
		})
	}
}

func TestTokenBudgetRejectsNonpositiveBeforeRequest(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, budget := range []int{0, -1} {
			t.Run(handler.name+"/"+map[bool]string{true: "zero", false: "negative"}[budget == 0], func(t *testing.T) {
				m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{text: `{"type":"person","name":"Alice"}`})
				_, err := handler.call(m, context.Background())
				if err == nil {
					t.Fatal("nonpositive budget was accepted")
				}
				requireTokenBudgetCalls(t, m, 0)
			})
		}
	}
}

func TestTokenBudgetStopsAtCumulativeThreshold(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, tc := range []struct {
			name     string
			budget   int
			attempts []tokenBudgetAttempt
			usage    UsageSum
		}{
			{"exact", 5, []tokenBudgetAttempt{{`{"name":}`, tokenBudgetUsage(3, 2)}}, *tokenBudgetUsage(3, 2)},
			{"cumulative", 9, []tokenBudgetAttempt{{`{"name":}`, tokenBudgetUsage(2, 2)}, {`{"name":}`, tokenBudgetUsage(3, 2)}}, *tokenBudgetUsage(5, 4)},
			{"over", 4, []tokenBudgetAttempt{{`{"name":}`, tokenBudgetUsage(3, 2)}}, *tokenBudgetUsage(3, 2)},
		} {
			t.Run(handler.name+"/"+tc.name, func(t *testing.T) {
				m := newTokenBudgetInstructor(&tc.budget, 5, tc.attempts...)
				response, err := handler.call(m, context.Background())
				requireTokenBudgetCalls(t, m, len(tc.attempts))
				requireTokenBudgetError(t, err, tc.budget, tc.usage)
				requireTokenBudgetResponseUsage(t, response, tc.usage)
			})
		}
	}
}

func TestTokenBudgetReturnsSuccessOverBudget(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, retry := range []bool{false, true} {
			name := "first_attempt"
			if retry {
				name = "after_retry"
			}
			t.Run(handler.name+"/"+name, func(t *testing.T) {
				budget := 5
				attempts := []tokenBudgetAttempt{{`{"type":"person","name":"Alice"}`, tokenBudgetUsage(8, 3)}}
				want := *tokenBudgetUsage(8, 3)
				if retry {
					attempts = append([]tokenBudgetAttempt{{`{"name":}`, tokenBudgetUsage(1, 1)}}, attempts...)
					want = *tokenBudgetUsage(9, 4)
				}
				m := newTokenBudgetInstructor(&budget, 2, attempts...)
				response, err := handler.call(m, context.Background())
				if err != nil {
					t.Fatalf("success must be returned even over budget: %v", err)
				}
				requireTokenBudgetCalls(t, m, len(attempts))
				requireTokenBudgetResponseUsage(t, response, want)
			})
		}
	}
}

func TestTokenBudgetUnknownUsageStopsRetry(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, tc := range []struct {
			name     string
			attempts []tokenBudgetAttempt
			want     UsageSum
		}{
			{"missing", []tokenBudgetAttempt{{text: `{"name":}`}}, UsageSum{}},
			{"partial", []tokenBudgetAttempt{{`{"name":}`, &UsageSum{InputTokens: 3}}}, UsageSum{InputTokens: 3}},
			{"partial_after_known", []tokenBudgetAttempt{{`{"name":}`, tokenBudgetUsage(2, 1)}, {`{"name":}`, &UsageSum{OutputTokens: 4}}}, UsageSum{InputTokens: 2, OutputTokens: 5, TotalTokens: 3}},
		} {
			t.Run(handler.name+"/"+tc.name, func(t *testing.T) {
				budget := 100
				m := newTokenBudgetInstructor(&budget, 5, tc.attempts...)
				response, err := handler.call(m, context.Background())
				requireTokenBudgetCalls(t, m, len(tc.attempts))
				requireTokenBudgetError(t, err, budget, tc.want)
				requireTokenBudgetResponseUsage(t, response, tc.want)
			})
		}
	}
}

func TestTokenBudgetKnownZeroUsageAllowsRetry(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		t.Run(handler.name, func(t *testing.T) {
			budget := 1
			m := newTokenBudgetInstructor(&budget, 1,
				tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(0, 0)},
				tokenBudgetAttempt{`{"type":"person","name":"Alice"}`, tokenBudgetUsage(1, 1)},
			)
			response, err := handler.call(m, context.Background())
			if err != nil {
				t.Fatalf("known zero usage should allow another request: %v", err)
			}
			requireTokenBudgetCalls(t, m, 2)
			requireTokenBudgetResponseUsage(t, response, *tokenBudgetUsage(1, 1))
		})
	}
}

func TestTokenBudgetWrapsLastFailure(t *testing.T) {
	budget := 3
	t.Run("json", func(t *testing.T) {
		m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(2, 1)})
		_, err := tokenBudgetHandlers()[0].call(m, context.Background())
		requireTokenBudgetError(t, err, budget, *tokenBudgetUsage(2, 1))
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("budget error must preserve JSON syntax error: %v", err)
		}
		requireTokenBudgetCalls(t, m, 1)
	})
	t.Run("validation", func(t *testing.T) {
		m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{`{"type":"person","name":""}`, tokenBudgetUsage(2, 1)})
		m.validate = true
		_, err := tokenBudgetHandlers()[0].call(m, context.Background())
		requireTokenBudgetError(t, err, budget, *tokenBudgetUsage(2, 1))
		var validationErr validator.ValidationErrors
		if !errors.As(err, &validationErr) || len(validationErr) != 1 || validationErr[0].Field() != "Name" {
			t.Fatalf("budget error must preserve Name validation error: %v", err)
		}
		requireTokenBudgetCalls(t, m, 1)
	})
	t.Run("union_discriminator", func(t *testing.T) {
		m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{`{"type":"unknown","name":"Alice"}`, tokenBudgetUsage(2, 1)})
		_, err := tokenBudgetHandlers()[1].call(m, context.Background())
		requireTokenBudgetError(t, err, budget, *tokenBudgetUsage(2, 1))
		var unionErr *UnionValidationError
		if !errors.As(err, &unionErr) || unionErr.DiscriminatorValue != "unknown" {
			t.Fatalf("budget error must preserve union discriminator failure: %v", err)
		}
		requireTokenBudgetCalls(t, m, 1)
	})
}

func TestTokenBudgetRetryLimitTakesPrecedence(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, usage := range []*UsageSum{nil, tokenBudgetUsage(2, 1)} {
			name := "missing_usage"
			if usage != nil {
				name = "exhausted_budget"
			}
			t.Run(handler.name+"/"+name, func(t *testing.T) {
				budget := 1
				m := newTokenBudgetInstructor(&budget, 0, tokenBudgetAttempt{`{"name":}`, usage})
				response, err := handler.call(m, context.Background())
				if err == nil {
					t.Fatal("expected retry limit error")
				}
				var budgetErr *TokenBudgetError
				if errors.As(err, &budgetErr) {
					t.Fatalf("budget must not override exhausted retry limit: %v", err)
				}
				requireTokenBudgetCalls(t, m, 1)
				want := UsageSum{}
				if usage != nil {
					want = *usage
				}
				requireTokenBudgetResponseUsage(t, response, want)
			})
		}
	}
}

func TestTokenBudgetContextPreventsRequests(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		for _, deadline := range []bool{false, true} {
			name := "canceled"
			if deadline {
				name = "deadline"
			}
			t.Run(handler.name+"/"+name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				want := context.Canceled
				if deadline {
					cancel()
					ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					want = context.DeadlineExceeded
				} else {
					cancel()
				}
				defer cancel()
				budget := 1
				m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(1, 1)})
				_, err := handler.call(m, ctx)
				if !errors.Is(err, want) {
					t.Fatalf("error = %v, want %v", err, want)
				}
				requireTokenBudgetCalls(t, m, 0)
			})
		}
		t.Run(handler.name+"/canceled_after_failure", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			budget := 1
			m := newTokenBudgetInstructor(&budget, 2, tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(2, 1)})
			m.after = cancel
			_, err := handler.call(m, ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("context cancellation must precede budget exhaustion: %v", err)
			}
			var budgetErr *TokenBudgetError
			if errors.As(err, &budgetErr) {
				t.Fatalf("cancellation was replaced by budget error: %v", err)
			}
			requireTokenBudgetCalls(t, m, 1)
		})
	}
}

func TestTokenBudgetDoesNotExtendRetryLimit(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		t.Run(handler.name, func(t *testing.T) {
			budget := 100
			m := newTokenBudgetInstructor(&budget, 1,
				tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(2, 1)},
				tokenBudgetAttempt{`{"name":}`, tokenBudgetUsage(3, 2)},
				tokenBudgetAttempt{`{"type":"person","name":"Alice"}`, tokenBudgetUsage(1, 1)},
			)
			response, err := handler.call(m, context.Background())
			if err == nil {
				t.Fatal("budget headroom must not extend retry limit")
			}
			var budgetErr *TokenBudgetError
			if errors.As(err, &budgetErr) {
				t.Fatalf("retry limit returned a budget error despite available budget: %v", err)
			}
			requireTokenBudgetCalls(t, m, 2)
			requireTokenBudgetResponseUsage(t, response, *tokenBudgetUsage(5, 3))
		})
	}
}

func TestTokenBudgetSuccessDoesNotRequireUsage(t *testing.T) {
	for _, handler := range tokenBudgetHandlers() {
		t.Run(handler.name, func(t *testing.T) {
			budget := 1
			m := newTokenBudgetInstructor(&budget, 2,
				tokenBudgetAttempt{text: `{"type":"person","name":"Alice"}`},
			)
			_, err := handler.call(m, context.Background())
			if err != nil {
				t.Fatalf("missing usage must not discard a successful result: %v", err)
			}
			requireTokenBudgetCalls(t, m, 1)
		})
	}
}
