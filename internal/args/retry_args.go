// Package args assembles environment and command arguments for sync and copy workers.
package args

import (
	"context"
	"time"

	"github.com/avast/retry-go"
)

// RetryArgs describes retry-go options derived from CLI or environment configuration.
type RetryArgs struct {
	// Attempts limits how many times an operation is retried.
	Attempts int
	// Delay sets the base delay between attempts.
	Delay time.Duration
	// MaxDelay switches delay handling to backoff when it exceeds Delay.
	MaxDelay time.Duration
	// MaxJitter caps random jitter added to delayed retries.
	MaxJitter time.Duration
}

// Assemble converts RetryArgs into retry-go options, including optional context cancellation.
func (a *RetryArgs) Assemble(ctx context.Context) []retry.Option {

	optionArgs := []retry.Option{
		retry.Attempts(uint(a.Attempts)),
	}

	if ctx != nil {
		optionArgs = append(optionArgs, retry.Context(ctx))
	}
	if a.Delay > 0 {
		optionArgs = append(optionArgs, retry.Delay(a.Delay))
		if a.MaxDelay > a.Delay {
			optionArgs = append(optionArgs,
				retry.MaxJitter(a.MaxJitter),
				retry.DelayType(retry.BackOffDelay),
			)
		} else {
			optionArgs = append(optionArgs, retry.DelayType(retry.FixedDelay))
		}
	}
	if a.MaxJitter > 0 {
		optionArgs = append(optionArgs, retry.MaxJitter(a.MaxJitter))
	}
	return optionArgs
}
