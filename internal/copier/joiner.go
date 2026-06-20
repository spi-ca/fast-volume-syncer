// Package copier batches scanned entries and sends them to the selected copy backend.
package copier

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type (
	// copyMethod copies one chunk of scanned entries.
	//
	// Implementations must treat the provided slice as read-only because the
	// chunk joiner may recycle the backing array after the call returns, and they
	// must tolerate concurrent calls from multiple chunk workers.
	copyMethod func(context.Context, []returns.Fileinfo) (returns.IOResult, error)

	// chunkJoiner groups scan results into bounded concurrent copy jobs.
	chunkJoiner struct {
		// taskSize limits how many chunk workers may run at once.
		taskSize int
		// chunkSize is the target number of entries per submitted chunk.
		chunkSize int

		// copier handles each submitted chunk.
		copier copyMethod
		// scanDuration flushes a partial chunk when scanning pauses.
		scanDuration time.Duration
	}

	// chunkResult carries one chunk worker result back to the runner.
	chunkResult struct {
		// Result holds per-chunk copy accounting.
		Result returns.IOResult
		// Error reports the chunk-level failure, if any.
		Error error
	}
)

// Execute starts the chunk dispatcher and returns a stream of chunk results.
func (c *chunkJoiner) Execute(ctx context.Context, entryRecvChan <-chan returns.Fileinfo) <-chan chunkResult {
	resultChan := make(chan chunkResult, c.taskSize)
	go c.dispatch(ctx, entryRecvChan, resultChan)
	return resultChan
}

// dispatch fills chunks from the scanner stream and submits them with bounded concurrency.
func (c *chunkJoiner) dispatch(ctx context.Context, entryRecvChan <-chan returns.Fileinfo, resultChan chan<- chunkResult) {
	sem := semaphore.NewWeighted(int64(c.taskSize))
	deadline := time.NewTicker(c.scanDuration)
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on chunkJoiner: %v", err)
		}
		_ = sem.Acquire(context.Background(), int64(c.taskSize))
		close(resultChan)
		deadline.Stop()
	}()

	chunkPool := sync.Pool{
		New: func() any {
			return make([]returns.Fileinfo, 0, c.chunkSize)
		},
	}

	taskCloser := func(chunk []returns.Fileinfo) {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on chunkHandler: %v", err)
		}
		sem.Release(1)
		chunkPool.Put(chunk[0:0])
	}

	var chunk []returns.Fileinfo
	for ended := false; !ended; {
		select {
		case <-ctx.Done():
			// 종료시 남은 항목은 무시한다.
			ended = true
			if chunk != nil {
				chunkPool.Put(chunk[0:0])
				chunk = nil
			}
			continue
		case entry, ok := <-entryRecvChan:
			if !ok {
				ended = true
				break
			}
			if chunk == nil {
				chunk = chunkPool.Get().([]returns.Fileinfo)[0:0]
			}
			chunk = append(chunk, entry)
			if len(chunk) < cap(chunk) {
				// busy loops
				continue
			} else {
				// full
				break
			}
		case <-deadline.C:
			break
		}

		if len(chunk) > 0 {
			// context문제가 있을때만 error 발생.
			if err := sem.Acquire(ctx, 1); err == nil {
				go c.submit(ctx, taskCloser, chunk, resultChan)
			} else {
				chunkPool.Put(chunk[0:0])
				ended = true
			}
			chunk = nil
		}
	}
}

// submit runs one copy job and forwards its result.
func (c *chunkJoiner) submit(ctx context.Context, closer func([]returns.Fileinfo), chunk []returns.Fileinfo, resultChan chan<- chunkResult) {
	defer closer(chunk)
	res := chunkResult{}
	res.Result, res.Error = c.copier(ctx, chunk)
	resultChan <- res
}
