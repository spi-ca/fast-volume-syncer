package util

import (
	"fmt"
	"os"
	"time"

	"github.com/slack-go/slack"
	"github.com/sony/gobreaker"
)

var (
	slackCircuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
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
	})
	slackWebhookUrl = ""
	hostname, _     = os.Hostname()
)

func SetSlackWebhookUrl(url string) {
	slackWebhookUrl = url
}

func SendSlackMessage(message string) {
	ErrLog.Printf(message)
	if len(slackWebhookUrl) == 0 {
		return
	}
	msg := slack.WebhookMessage{
		Text: fmt.Sprintf("[%s]%s:%s", hostname, InfoLog.Prefix(), message),
	}

	err := slack.PostWebhook(slackWebhookUrl, &msg)
	if err != nil {
		ErrLog.Printf("failed to send message: %v", err)
	}
}
