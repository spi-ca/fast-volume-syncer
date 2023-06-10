package selector

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type workerJoiner struct {
	wg  sync.WaitGroup
	sem chan bool

	invoker *Invoker
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

func (c *workerJoiner) dispatch(ctx context.Context, entryRecvChan <-chan copyEntry, errorChan chan<- error) {
	//ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on workerJoiner: %v", err)
		}
		//cancel()
		c.wg.Wait()
		close(errorChan)
	}()

	for {
		select {
		case <-ctx.Done():
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

	started := time.Now()
	err := c.invoker.Run(ctx, entry)
	ended := time.Now()
	util.InfoLog.Printf("copyEntry completed in %2.2f ms", float32(ended.Sub(started).Microseconds())/1000)
	if err != nil {
		errorChan <- err
	} else {
		util.SendSlackMessage(fmt.Sprintf("복사항목 복사 완료 %s, 소요시간 %s", entry, ended.Sub(started)))
	}
}
