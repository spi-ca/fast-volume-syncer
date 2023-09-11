package returns

import (
	"time"
)

type IOResult interface {
	Duration() time.Duration

	Total() int64
	Files() int64
	Links() int64
	Directories() int64

	SentBytes() int64
}
