package util

import (
	"fmt"
	"os"
	"strings"
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
	loggerPrefix    = "[slack]"
	maxMessageLen   = 40000
)

var (
	SlackSender = &slackSender{
		circuitBreaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "slack",
			MaxRequests: 100,
			Interval:    500 * time.Millisecond,
			Timeout:     30 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := counts.TotalFailures
				failureRatio *= 100
				failureRatio /= counts.Requests
				return counts.Requests >= 3 && failureRatio >= 60
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				if from == gobreaker.StateClosed && to == gobreaker.StateOpen {
					ErrLog.Printf("%sendpoint unavailable", loggerPrefix)
				} else if from == gobreaker.StateHalfOpen && to == gobreaker.StateClosed {
					ErrLog.Printf("%sendpoint is returning available", loggerPrefix)
				}
			},
		}),
		retry: args.RetryArgs{
			Attempts:  15,
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
	doneChan       <-chan struct{}
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
	if s.doneChan != nil {
		return
	}

	hostname, _ := os.Hostname()
	messageChan := make(chan string, 100)
	doneChan := make(chan struct{})
	s.hostname = hostname
	go s.senderLoop(messageChan, doneChan)
	s.messageChan = messageChan
	s.doneChan = doneChan
}

func (s *slackSender) Close() {
	if s.messageChan == nil {
		return
	}
	s.m.Lock()
	defer s.m.Unlock()
	close(s.messageChan)
	<-s.doneChan
	s.messageChan = nil
	s.doneChan = nil
}

func (s *slackSender) Write(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, nil
	}
	s.m.RLock()
	defer s.m.RUnlock()

	if s.messageChan != nil {
		s.messageChan <- string(b)
	}
	return len(b), nil
}

func (s *slackSender) senderLoop(msgChan <-chan string, doneChan chan<- struct{}) {
	deadline := time.NewTicker(5 * time.Second)
	defer func() {
		if err := recover(); err != nil {
			ErrLog.Printf("%spanic on slackSender.senderLoop: %v", loggerPrefix, err)
		}
		deadline.Stop()
		close(doneChan)
	}()

	prefix := fmt.Sprintf("[%s]", s.hostname)
	msgContainer := slack.WebhookMessage{}
	retryArgs := s.retry.Assemble(nil)
	senderFunc := func() (any, error) {
		return nil, slack.PostWebhook(s.webhookUrl, &msgContainer)
	}

	retryFunc := func() error {
		_, err := s.circuitBreaker.Execute(senderFunc)
		if err == nil {
			return nil
		} else if retryable, ok := err.(slackutilsx.Retryable); !ok || !retryable.Retryable() {
			return retry.Unrecoverable(err)
		}

		if rateLimitedErr, ok := err.(*slack.RateLimitedError); ok {
			<-time.After(rateLimitedErr.RetryAfter)
		}
		return err
	}
	var remainMessage = ""
	var builder = strings.Builder{}
	for ended := false; !ended; {
		select {
		case entry, ok := <-msgChan:
			if !ok {
				ended = true
				break
			} else if strings.HasPrefix(entry, loggerPrefix) {
				continue
			} else if builder.Len() > maxMessageLen {
				remainMessage = entry
				break
			}
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(prefix)
			builder.WriteString(entry)
			continue
		case <-deadline.C:
			break
		}

		if builder.Len() > 0 {
			msgContainer.Text = builder.String()
			if err := retry.Do(retryFunc, retryArgs...); err != nil {
				ErrLog.Printf("%sfailed to send message: %v", loggerPrefix, err)
			}
			builder.Reset()
		}

		if len(remainMessage) > 0 {
			builder.WriteString(prefix)
			builder.WriteString(remainMessage)
			msgContainer.Text = builder.String()
			if err := retry.Do(retryFunc, retryArgs...); err != nil {
				ErrLog.Printf("%sfailed to send message: %v", loggerPrefix, err)
			}
			remainMessage = ""
			builder.Reset()
		}
	}
}
