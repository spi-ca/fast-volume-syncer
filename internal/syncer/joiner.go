package syncer

import (
	"context"
	"sync"
	"time"

	"go.uber.org/multierr"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/model"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type chunkJoiner struct {
	wg  sync.WaitGroup
	sem chan bool

	errorChan chan error

	chunkPool sync.Pool

	invoker      *rsync.Task
	scanDuration time.Duration
}

func newChunkJoiner(
	taskSize int, chunkSize int,
	scanDuration time.Duration,
	invoker *rsync.Task,
) *chunkJoiner {
	return &chunkJoiner{
		sem: make(chan bool, taskSize),

		errorChan: make(chan error, taskSize),

		invoker: invoker,
		chunkPool: sync.Pool{
			New: func() interface{} {
				return make([]model.Fileinfo, 0, chunkSize)
			},
		},
		scanDuration: scanDuration,
	}
}
func (c *chunkJoiner) Execute(ctx context.Context, entryRecvChan <-chan model.Fileinfo) error {
	go c.dispatchChunks(ctx, entryRecvChan)

	var err error
	for newErr := range c.errorChan {
		err = multierr.Append(err, newErr)
	}
	return err
}

func (c *chunkJoiner) dispatchChunks(parentCtx context.Context, entryRecvChan <-chan model.Fileinfo) {
	ended := false
	deadline := time.NewTicker(c.scanDuration)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on chunkJoiner: %v", err)
		}
		c.wg.Wait()
		close(c.errorChan)
		cancel()
		deadline.Stop()
	}()

	var chunk []model.Fileinfo
	for !ended {
		select {
		case <-parentCtx.Done():
			// 종료시 남은 항목은 무시한다.
			ended = true
			if chunk != nil {
				c.chunkPool.Put(chunk[0:0])
			}
			continue
		case entry, ok := <-entryRecvChan:
			if !ok {
				ended = true
				break
			}
			if chunk == nil {
				chunk = c.chunkPool.Get().([]model.Fileinfo)[0:0]
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
			go c.submit(ctx, chunk)
			chunk = nil
		}
		if ended {
			break
		}
	}
}

func (c *chunkJoiner) submit(ctx context.Context, chunk []model.Fileinfo) {
	defer func() {
		<-c.sem
		c.wg.Done()
		c.chunkPool.Put(chunk[0:0])
	}()
	err := c.invoker.Execute(ctx, chunk)
	if err != nil {
		c.errorChan <- err
	}
}
