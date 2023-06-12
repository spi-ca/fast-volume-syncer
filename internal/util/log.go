package util

import (
	log "github.com/sirupsen/logrus"
	"io"
	glog "log"
	"os"
)

var (
	LogFormatter = &log.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "060102150405",
	}
)

func init() {
	log.SetFormatter(LogFormatter)
	log.SetOutput(io.Discard)
	log.SetLevel(log.InfoLevel)
	glog.SetOutput(log.StandardLogger().Writer())
	log.AddHook(&WriterHook{
		Writer: os.Stderr,
		LogLevels: []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		},
	})
	log.AddHook(&WriterHook{
		Writer: os.Stdout,
		LogLevels: []log.Level{
			log.InfoLevel,
			log.DebugLevel,
		},
	})
}

// WriterHook is a hook that writes logs of specified LogLevels to specified Writer
type WriterHook struct {
	Writer    io.Writer
	LogLevels []log.Level
}

// Fire will be called when some logging function is called with current hook
// It will format log entry to string and write it to appropriate writer
func (hook *WriterHook) Fire(entry *log.Entry) error {
	line, err := entry.Bytes()
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write(line)
	return err
}

// Levels define on which log levels this hook would trigger
func (hook *WriterHook) Levels() []log.Level {
	return hook.LogLevels
}
