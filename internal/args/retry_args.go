package args

import (
	"context"
	"time"

	"github.com/avast/retry-go"
)

type RetryArgs struct {
	Attempts  int
	Delay     time.Duration
	MaxDelay  time.Duration
	MaxJitter time.Duration
}

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
