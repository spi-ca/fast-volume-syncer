package syncer

import (
	"context"
	"log"
	"sync"
	"time"

	"go.uber.org/multierr"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
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
				return make([]common.Fileinfo, 0, chunkSize)
			},
		},
		scanDuration: scanDuration,
	}
}

func (c *chunkJoiner) dispatchChunks(ctx context.Context, entryRecvChan <-chan common.Fileinfo) {
	ended := false
	deadline := time.NewTicker(c.scanDuration)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic on chunkJoiner: %v", err)
		}

		log.Printf("closing chunk channel")
		close(c.errorChan)
		deadline.Stop()
	}()

	var chunk []common.Fileinfo
	for !ended {
		select {
		case entry, ok := <-entryRecvChan:
			if !ok {
				ended = true
				break
			}
			if chunk == nil {
				chunk = c.chunkPool.Get().([]common.Fileinfo)[0:0]
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

		// 남은것 처리
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
func (c *chunkJoiner) Execute(ctx context.Context, entryRecvChan <-chan common.Fileinfo) error {
	go c.dispatchChunks(ctx, entryRecvChan)

	var err error
	for newErr := range c.errorChan {
		err = multierr.Append(err, newErr)
	}
	c.wg.Wait()

	return err
}

func (c *chunkJoiner) submit(ctx context.Context, chunk []common.Fileinfo) {
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
