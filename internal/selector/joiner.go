package selector

import (
	"context"
	"sync"

	"go.uber.org/multierr"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type workerJoiner struct {
	wg  sync.WaitGroup
	sem chan bool

	errorChan chan error

	invoker *Invoker
}

func newWorkerJoiner(workerSize int, invoker *Invoker) *workerJoiner {
	return &workerJoiner{
		sem:       make(chan bool, workerSize),
		errorChan: make(chan error, workerSize),
		invoker:   invoker,
	}
}

func (c *workerJoiner) Execute(ctx context.Context, entryRecvChan <-chan copyEntry) error {
	go c.dispatch(ctx, entryRecvChan)

	var err error
	for newErr := range c.errorChan {
		err = multierr.Append(err, newErr)
	}
	return err
}

func (c *workerJoiner) dispatch(parentCtx context.Context, entryRecvChan <-chan copyEntry) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on workerJoiner: %v", err)
		}
		c.wg.Wait()
		close(c.errorChan)
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
			go c.submit(ctx, entry)
		}
	}
}

func (c *workerJoiner) submit(ctx context.Context, entry copyEntry) {
	defer func() {
		<-c.sem
		c.wg.Done()
	}()
	if err := c.invoker.Run(ctx, entry); err != nil {
		c.errorChan <- err
	}
}
