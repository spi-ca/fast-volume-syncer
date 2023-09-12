package returns

type IOResult interface {
	Total() int64
	Files() int64
	Links() int64
	Directories() int64

	SentBytes() int64
}
