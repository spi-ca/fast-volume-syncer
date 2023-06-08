package slack

//
//import (
//	"bufio"
//	"bytes"
//	"context"
//	"github.com/pkg/errors"
//	"github.com/sony/gobreaker"
//	"go.uber.org/zap"
//	"go.uber.org/zap/zapcore"
//	"net/http"
//	"runtime"
//	"strings"
//	"sync/atomic"
//	"time"
//)
//
//const (
//	logsender      = "slack"
//	default_sender = "default"
//)
//
//type slackSender struct {
//	urls map[string][]byte
//	host string
//
//	logger *zap.SugaredLogger
//
//	msgInHandler  *util.Handler
//	msgOutHandler *util.Handler
//
//	unsent int32
//
//	msgSendDoneChan chan struct{}
//	msgInChan       chan<- SlackMessage
//	msgOutChan      <-chan SlackMessage
//	breaker         *gobreaker.CircuitBreaker
//
//	clientGetter func(concurrency int) *http.Client
//	_            struct{}
//}
//
//func (ls *slackSender) StartHandling() {
//	ls.msgSendDoneChan = make(chan struct{})
//	ls.msgInHandler = util.NewHandler(context.Background())
//	ls.msgOutHandler = util.NewHandler(context.Background())
//	ls.msgInChan, ls.msgOutChan = newSendQueue()
//	ls.logger = zlog.New(nil, logsender)
//
//	ls.breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
//		Name:        "slack",
//		MaxRequests: 100,
//		Interval:    500 * time.Millisecond,
//		Timeout:     3 * time.Second,
//		ReadyToTrip: func(counts gobreaker.Counts) bool {
//			failureRatio := counts.TotalFailures
//			failureRatio *= 100
//			failureRatio /= counts.Requests
//			return counts.Requests >= 3 && failureRatio >= 60
//		},
//		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
//			if from == gobreaker.StateClosed && to == gobreaker.StateOpen {
//				ls.logger.Warn("endpoint unavailable")
//			} else if from == gobreaker.StateHalfOpen && to == gobreaker.StateClosed {
//				ls.logger.Warn("endpoint is returning available")
//			}
//		},
//	})
//
//	concurrency := runtime.NumCPU()
//	for i := 0; i < concurrency; i++ {
//		ls.msgOutHandler.IncreaseWait()
//		go ls.consumeLoop(i)
//	}
//}
//
///*
//*
//원격 로거를 잘 종료한다.
//*/
//func (ls *slackSender) StopHandling() {
//	if ls.msgOutHandler != nil {
//		ls.logger.Info("slackSender: draining...")
//		close(ls.msgSendDoneChan)
//
//		ls.msgInHandler.GracefulWait()
//		close(ls.msgInChan)
//		ls.msgOutHandler.GracefulWait()
//
//		if unsent := atomic.LoadInt32(&ls.unsent); unsent > 0 {
//			ls.logger.Errorf("slackSender: unsent errors :%d", unsent)
//		}
//
//		ls.logger.Info("slackSender: stopped slackSender handling")
//		ls.msgOutHandler = nil
//
//	}
//}
//
//func (ls *slackSender) consumeLoop(idx int) {
//	defer func() {
//		ls.logger.Debugf("slackSender: sender(%d) closed!", idx)
//		ls.msgOutHandler.DecreaseWait()
//	}()
//	ls.logger.Debugf("slackSender: sender(%d) ready", idx)
//	sender := ls.clientGetter(5)
//	readBuf := bufio.NewReaderSize(nil, 1024)
//	for entry := range ls.msgOutChan {
//		if err := ls.internalSend(sender, readBuf, &entry); err != nil {
//			ls.logger.Error("slackSender: ", err.Error())
//			atomic.AddInt32(&ls.unsent, 1)
//		}
//	}
//
//}
//func (ls *slackSender) internalSend(sender *fasthttp.Client, readBuf *bufio.Reader, entry *SlackMessage) (err error) {
//	sendUrl, ok := ls.urls[entry.Channel]
//	if !ok {
//		err = errors.Errorf("channel(%s) not fonund!", entry.Channel)
//		return
//	}
//
//	stream := myjson.JSONConfig.BorrowStream(nil)
//	defer myjson.JSONConfig.ReturnStream(stream)
//
//	if stream.WriteVal(entry); stream.Error != nil {
//		err = stream.Error
//		return
//	}
//
//	//
//	_, err = ls.breaker.Execute(func() (ret interface{}, internalErr error) {
//		req := fasthttp.AcquireRequest()
//		defer fasthttp.ReleaseRequest(req)
//
//		req.Header.SetMethod(http.MethodPost)
//		req.Header.SetContentType(myhttp.JsonContentUTF8Type[0])
//		req.SetBodyStream(bytes.NewReader(stream.Buffer()), stream.Buffered())
//		req.SetRequestURIBytes(sendUrl)
//
//		resp := fasthttp.AcquireResponse()
//		defer fasthttp.ReleaseResponse(resp)
//
//		if internalErr = sender.Do(req, resp); internalErr != nil {
//			//do nothing
//		} else if statusCode := resp.StatusCode(); statusCode == http.StatusOK {
//			//do nothing
//		} else if readBuf == nil {
//			internalErr = errors.New(http.StatusText(statusCode))
//			ls.logger.Errorf(
//				"slack send error: status(%d)\n\tBODY:%s",
//				statusCode,
//				gotils.B2S(resp.Body()),
//			)
//		} else {
//			internalErr = errors.New(http.StatusText(statusCode))
//			defer readBuf.Reset(nil)
//			var readBufData []byte
//			if readErr := resp.Read(readBuf); readErr != nil {
//				ls.logger.Errorf(
//					"slack send error: status(%d), read err: %s",
//					statusCode,
//					readErr.Error(),
//				)
//			} else if readBufData, readErr = readBuf.Peek(readBuf.Buffered()); readErr != nil {
//				ls.logger.Errorf(
//					"slack send error: status(%d), peek err: %s",
//					statusCode,
//					readErr.Error(),
//				)
//			} else {
//				ls.logger.Errorf(
//					"slack send error: status(%d)\n\tBODY:%s",
//					statusCode,
//					gotils.B2S(readBufData),
//				)
//			}
//		}
//		return
//
//	})
//	return
//}
//
//// for zap log hook
//func (ls *slackSender) Send(entry zapcore.Entry) (err error) {
//	if entry.Level < zapcore.WarnLevel {
//		return
//	} else if strings.HasPrefix(entry.LoggerName, logsender) {
//		return
//	}
//
//	select {
//	case <-ls.msgSendDoneChan:
//		atomic.AddInt32(&ls.unsent, 1)
//		return
//	default:
//		ls.msgInHandler.IncreaseWait()
//		defer ls.msgInHandler.DecreaseWait()
//	}
//
//	builder := ls.startMessage()
//	builder.WriteRune('/')
//	builder.WriteString(entry.Level.CapitalString()[:1])
//	builder.WriteRune('[')
//	builder.WriteString(entry.LoggerName)
//	builder.WriteRune(']')
//	builder.WriteRune(' ')
//	builder.WriteString(entry.Message)
//
//	ls.msgInChan <- SlackMessage{
//		Parse:     "full",
//		Timestamp: myjson.JSONTime(entry.Time),
//		Text:      builder.String(),
//		Channel:   default_sender,
//	}
//	return
//}
//
//func (ls *slackSender) SendMessage(msg SlackMessage) (err error) {
//	select {
//	case <-ls.msgSendDoneChan:
//		atomic.AddInt32(&ls.unsent, 1)
//		return
//	default:
//		ls.msgInHandler.IncreaseWait()
//		defer ls.msgInHandler.DecreaseWait()
//	}
//	ls.msgInChan <- msg
//	return
//}
//
//func (ls *slackSender) startMessage() *strings.Builder {
//	builder := &strings.Builder{}
//	builder.WriteRune('[')
//	builder.WriteString(ls.host)
//	builder.WriteRune(']')
//	return builder
//}
//
//func (ls *slackSender) Test(message string) (err error) {
//	ls.msgInHandler.IncreaseWait()
//	defer ls.msgInHandler.DecreaseWait()
//
//	builder := ls.startMessage()
//	builder.WriteString(message)
//
//	msg := SlackMessage{
//		Parse:   "full",
//		Text:    builder.String(),
//		Channel: default_sender,
//	}
//	sender := ls.clientGetter(1)
//
//	if err = ls.internalSend(sender, nil, &msg); err != nil {
//		ls.logger.Error("slackSender: ", err.Error())
//		atomic.AddInt32(&ls.unsent, 1)
//	}
//	return
//}
