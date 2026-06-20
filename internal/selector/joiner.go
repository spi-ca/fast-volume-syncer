// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// workerJoiner bounds concurrent sync children and waits for all submitted work to finish.
type workerJoiner struct {
	// workerSize is the maximum number of sync children allowed to run concurrently.
	workerSize int
	// invoker starts the sync child process for each accepted CSV entry.
	invoker *Invoker
}

// Execute starts the fan-out loop and returns a stream of worker errors.
func (c *workerJoiner) Execute(ctx context.Context, entryRecvChan <-chan copyEntry) <-chan error {
	errorChan := make(chan error, c.workerSize)
	go c.dispatch(ctx, entryRecvChan, errorChan)
	return errorChan
}

// dispatch reads parsed CSV entries, acquires worker slots, and submits each sync child.
func (c *workerJoiner) dispatch(ctx context.Context, entryRecvChan <-chan copyEntry, errorChan chan<- error) {
	sem := semaphore.NewWeighted(int64(c.workerSize))
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on workerJoiner: %v", err)
		}
		_ = sem.Acquire(context.Background(), int64(c.workerSize))
		close(errorChan)
	}()

	workerCloser := func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on worker: %v", err)
		}
		sem.Release(1)
	}

	for entry := range entryRecvChan {
		if err := sem.Acquire(ctx, 1); err == nil {
			go c.submit(ctx, workerCloser, entry, errorChan)
		} else {
			return
		}
	}
}

// submit runs one entry through the invoker and reports its duration or failure.
func (c *workerJoiner) submit(ctx context.Context, closer func(), entry copyEntry, errorChan chan<- error) {
	defer closer()

	started := time.Now()
	err := c.invoker.Run(ctx, entry)
	ended := time.Now()
	if err != nil {
		errorChan <- fmt.Errorf("%s failed in %s: %w", entry, ended.Sub(started), err)
	} else {
		util.ErrLog.Printf("%s completed in %s", entry, ended.Sub(started))
	}
}
