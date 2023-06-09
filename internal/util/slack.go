package util

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/avast/retry-go"

	"github.com/slack-go/slack/slackutilsx"

	"github.com/slack-go/slack"
	"github.com/sony/gobreaker"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
)

const (
	slackWebhookUrl = "https://example.invalid/webhook"
)

var (
	SlackSender = &slackSender{
		circuitBreaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "slack",
			MaxRequests: 100,
			Interval:    500 * time.Millisecond,
			Timeout:     3 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := counts.TotalFailures
				failureRatio *= 100
				failureRatio /= counts.Requests
				return counts.Requests >= 3 && failureRatio >= 60
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				if from == gobreaker.StateClosed && to == gobreaker.StateOpen {
					ErrLog.Print("endpoint unavailable")
				} else if from == gobreaker.StateHalfOpen && to == gobreaker.StateClosed {
					ErrLog.Print("endpoint is returning available")
				}
			},
		}),
		retry: args.RetryArgs{
			Attempts:  3,
			Delay:     5 * time.Second,
			MaxDelay:  1 * time.Minute,
			MaxJitter: 15 * time.Second,
		},
		webhookUrl: slackWebhookUrl,
	}
)

type slackSender struct {
	circuitBreaker *gobreaker.CircuitBreaker
	hostname       string
	webhookUrl     string
	messageChan    chan<- string
	retry          args.RetryArgs
	m              sync.RWMutex
}

func (s *slackSender) Send(message string) {
	ErrLog.Printf(message)
	s.m.RLock()
	defer s.m.RUnlock()
	if s.messageChan == nil {
		return
	}
	s.messageChan <- message
}

func (s *slackSender) Start() {
	s.m.Lock()
	defer s.m.Unlock()
	hostname, _ := os.Hostname()
	messageChan := make(chan string, 100)
	s.hostname = hostname
	go s.senderLoop(messageChan)
	s.messageChan = messageChan
}

func (s *slackSender) Close() {
	if s.messageChan == nil {
		return
	}
	close(s.messageChan)
	s.m.Lock()
	defer s.m.Unlock()
	s.messageChan = nil
}

func (s *slackSender) senderLoop(msgChan <-chan string) {
	s.m.RLock()
	defer s.m.RUnlock()

	msgContainer := slack.WebhookMessage{}
	retryArgs := s.retry.Assemble(nil)

	senderFunc := func() (any, error) {
		err := slack.PostWebhook(s.webhookUrl, &msgContainer)
		if err == nil {
			return nil, nil
		} else if retryable, ok := err.(slackutilsx.Retryable); !ok || !retryable.Retryable() {
			return nil, retry.Unrecoverable(err)
		}
		if rateLimitedErr, ok := err.(*slack.RateLimitedError); ok {
			<-time.After(rateLimitedErr.RetryAfter)
		}
		return nil, err
	}

	processFunc := func() error {
		_, err := s.circuitBreaker.Execute(senderFunc)
		return err
	}

	if s.retry.Attempts > 0 {
		retryFunc := processFunc
		processFunc = func() error {
			return retry.Do(retryFunc, retryArgs...)
		}
	}

	for msg := range msgChan {
		msgContainer.Text = fmt.Sprintf("[%s]%s:%s", s.hostname, InfoLog.Prefix(), msg)
		if err := processFunc(); err != nil {
			ErrLog.Printf("failed to send message: %v", err)
		}
	}
}

func SendSlackMessage(message string) {
	SlackSender.Send(message)
}
