package core

import (
	"context"
	"fmt"
)

// TokenBudgetError includes usage from failed attempts and wraps the last error.
type TokenBudgetError struct {
	Budget     int
	Usage      UsageSum
	UsageKnown bool
	Err        error
}

func (e *TokenBudgetError) Error() string {
	if !e.UsageKnown {
		return fmt.Sprintf("cannot retry with token budget %d: token usage is missing or incomplete: %v", e.Budget, e.Err)
	}
	return fmt.Sprintf("token budget exhausted (%d >= %d): %v", e.Usage.TotalTokens, e.Budget, e.Err)
}

func (e *TokenBudgetError) Unwrap() error { return e.Err }

func retryTokenBudget(i Instructor) (int, error) {
	provider, ok := i.(interface{ TokenBudget() *int })
	if !ok {
		return 0, nil
	}
	budget := provider.TokenBudget()
	if budget == nil {
		return 0, nil
	}
	if *budget <= 0 {
		return 0, fmt.Errorf("token budget must be positive, got %d", *budget)
	}
	return *budget, nil
}

func checkRetryTokenBudget(ctx context.Context, budget int, usage *UsageSum, lastErr error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if budget == 0 || (usage.UsageKnown && usage.TotalTokens < budget) {
		return nil
	}
	return &TokenBudgetError{
		Budget: budget, Usage: *usage, UsageKnown: usage.UsageKnown, Err: lastErr,
	}
}
