package syncer

import (
	"context"
	"errors"
	"sync"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type chunkJoiner struct {
	wg  sync.WaitGroup
	sem chan bool

	entryRecvChan <-chan returns.Fileinfo

	chunkPool sync.Pool

	invoker      *rsync.Task
	scanDuration time.Duration
}

func (c *chunkJoiner) Execute(ctx context.Context) error {
	errorChan := make(chan error, len(c.sem))
	go c.dispatch(ctx, errorChan)

	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *chunkJoiner) dispatch(parentCtx context.Context, errorChan chan<- error) {
	ended := false
	deadline := time.NewTicker(c.scanDuration)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on chunkJoiner: %v", err)
		}
		c.wg.Wait()
		close(errorChan)
		cancel()
		deadline.Stop()
	}()

	var chunk []returns.Fileinfo
	for !ended {
		select {
		case <-parentCtx.Done():
			// 종료시 남은 항목은 무시한다.
			ended = true
			if chunk != nil {
				c.chunkPool.Put(chunk[0:0])
			}
			continue
		case entry, ok := <-c.entryRecvChan:
			if !ok {
				ended = true
				break
			}
			if chunk == nil {
				chunk = c.chunkPool.Get().([]returns.Fileinfo)[0:0]
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
			c.wg.Add(1)
			c.sem <- true
			go c.submit(ctx, chunk, errorChan)
			chunk = nil
		}
		if ended {
			break
		}
	}
}

func (c *chunkJoiner) submit(ctx context.Context, chunk []returns.Fileinfo, errorChan chan<- error) {
	defer func() {
		<-c.sem
		c.wg.Done()
		c.chunkPool.Put(chunk[0:0])
	}()
	err := c.invoker.Execute(ctx, chunk)
	if err != nil {
		errorChan <- err
	}
}
