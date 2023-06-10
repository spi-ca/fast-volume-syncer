package selector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type workerJoiner struct {
	workerSize int
	invoker    *Invoker
}

func (c *workerJoiner) Execute(ctx context.Context, entryRecvChan <-chan copyEntry) error {
	errorChan := make(chan error, c.workerSize)
	go c.dispatch(ctx, entryRecvChan, errorChan)

	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *workerJoiner) dispatch(ctx context.Context, entryRecvChan <-chan copyEntry, errorChan chan<- error) {
	sem := semaphore.NewWeighted(int64(c.workerSize))
	defer func() {
		_ = sem.Acquire(context.Background(), int64(c.workerSize))
		close(errorChan)
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on workerJoiner: %v", err)
		}
	}()

	workerCloser := func() {
		sem.Release(1)
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on worker: %v", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			// 종료시 남은 항목은 무시한다.
			return
		case entry, ok := <-entryRecvChan:
			if !ok {
				return
			}
			_ = sem.Acquire(ctx, 1)
			go c.submit(ctx, workerCloser, entry, errorChan)
		}
	}
}

func (c *workerJoiner) submit(ctx context.Context, closer func(), entry copyEntry, errorChan chan<- error) {
	defer closer()

	started := time.Now()
	err := c.invoker.Run(ctx, entry)
	ended := time.Now()
	util.InfoLog.Printf("copyEntry completed in %2.2f ms", float32(ended.Sub(started).Microseconds())/1000)
	if err != nil {
		errorChan <- err
	} else {
		util.SendSlackMessage(fmt.Sprintf("copyEntry(%s) completed in %s", entry, ended.Sub(started)))
	}
}
