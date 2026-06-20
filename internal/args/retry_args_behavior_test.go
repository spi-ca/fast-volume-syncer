// Package args assembles environment and command arguments for sync and copy workers.
package args

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avast/retry-go"
)

// TestRetryArgsAssembleHonorsAttempts verifies the assembled options stop after the configured attempt count.
func TestRetryArgsAssembleHonorsAttempts(t *testing.T) {
	attempts := 0
	options := (&RetryArgs{Attempts: 3}).Assemble(context.Background())
	err := retry.Do(func() error {
		attempts++
		return errors.New("try again")
	}, options...)
	if err == nil {
		t.Fatal("expected retry.Do to return the final error")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

// TestRetryArgsAssembleHonorsCanceledContext verifies cancellation is forwarded through retry.Context.
func TestRetryArgsAssembleHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	options := (&RetryArgs{Attempts: 5, Delay: time.Millisecond}).Assemble(ctx)
	err := retry.Do(func() error {
		attempts++
		return errors.New("try again")
	}, options...)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts > 1 {
		t.Fatalf("attempts after canceled context = %d, want at most 1", attempts)
	}
}

// BenchmarkRetryArgsAssemble measures allocation cost while building retry-go options.
func BenchmarkRetryArgsAssemble(b *testing.B) {
	ctx := context.Background()
	args := &RetryArgs{Attempts: 3, Delay: time.Millisecond, MaxDelay: 10 * time.Millisecond, MaxJitter: time.Millisecond}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if len(args.Assemble(ctx)) == 0 {
			b.Fatal("expected retry options")
		}
	}
}
