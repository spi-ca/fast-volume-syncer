package selector

import (
	"context"
	"errors"
	"sync"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type workerJoiner struct {
	wg  sync.WaitGroup
	sem chan bool

	invoker *Invoker
}

func newWorkerJoiner(workerSize int, invoker *Invoker) *workerJoiner {
	return &workerJoiner{
		sem:     make(chan bool, workerSize),
		invoker: invoker,
	}
}

func (c *workerJoiner) Execute(ctx context.Context, entryRecvChan <-chan copyEntry) error {
	errorChan := make(chan error, len(c.sem))
	go c.dispatch(ctx, entryRecvChan, errorChan)

	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *workerJoiner) dispatch(parentCtx context.Context, entryRecvChan <-chan copyEntry, errorChan chan<- error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on workerJoiner: %v", err)
		}
		c.wg.Wait()
		close(errorChan)
		cancel()
	}()

	for {
		select {
		case <-parentCtx.Done():
			// 종료시 남은 항목은 무시한다.
			return
		case entry, ok := <-entryRecvChan:
			if !ok {
				return
			}
			c.wg.Add(1)
			c.sem <- true
			go c.submit(ctx, entry, errorChan)
		}
	}
}

func (c *workerJoiner) submit(ctx context.Context, entry copyEntry, errorChan chan<- error) {
	defer func() {
		<-c.sem
		c.wg.Done()
	}()
	if err := c.invoker.Run(ctx, entry); err != nil {
		errorChan <- err
	}
}
