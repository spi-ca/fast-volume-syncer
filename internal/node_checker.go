package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func NewNodeClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
		Timeout: 5 * time.Second,
	}
}

func NodeStatusChecker(ctx context.Context, client *http.Client, expectedStatus NodeStatus, errorChan chan<- error) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic on nodeStatusChecker: %v", err)
		}
		close(errorChan)
	}()

	i := &struct {
		Status NodeStatus `json:"state"`
	}{}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resp, err := client.Get("http://localhost/api/v1/vm.info")
		if err != nil {
			errorChan <- err
			return
		} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errorChan <- fmt.Errorf("http error(%d): %s", resp.StatusCode, http.StatusText(resp.StatusCode))
			return
		}

		d := json.NewDecoder(resp.Body)
		err = d.Decode(&i)
		if err != nil {
			errorChan <- err
			return
		} else if i.Status != expectedStatus {
			errorChan <- fmt.Errorf("failed status %s, expected: %s", i.Status, expectedStatus)
			return
		} else {
			_ = <-ticker.C
		}
	}
}
