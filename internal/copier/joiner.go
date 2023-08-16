package copier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type (
	copyMethod  func(context.Context, []returns.Fileinfo) error
	chunkJoiner struct {
		taskSize  int
		chunkSize int

		copier       copyMethod
		scanDuration time.Duration
	}
)

func (c *chunkJoiner) Execute(ctx context.Context, entryRecvChan <-chan returns.Fileinfo) <-chan error {
	errorChan := make(chan error, c.taskSize)
	go c.dispatch(ctx, entryRecvChan, errorChan)
	return errorChan
}

func (c *chunkJoiner) dispatch(ctx context.Context, entryRecvChan <-chan returns.Fileinfo, errorChan chan<- error) {
	sem := semaphore.NewWeighted(int64(c.taskSize))
	deadline := time.NewTicker(c.scanDuration)
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on chunkJoiner: %v", err)
		}
		_ = sem.Acquire(context.Background(), int64(c.taskSize))
		close(errorChan)
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
				go c.submit(ctx, taskCloser, chunk, errorChan)
			} else {
				chunkPool.Put(chunk[0:0])
				ended = true
			}
			chunk = nil
		}
	}
}

func (c *chunkJoiner) submit(ctx context.Context, closer func([]returns.Fileinfo), chunk []returns.Fileinfo, errorChan chan<- error) {
	defer closer(chunk)
	err := c.copier(ctx, chunk)
	if err != nil {
		errorChan <- fmt.Errorf("chunk processing failed : %w", err)
	}
}
