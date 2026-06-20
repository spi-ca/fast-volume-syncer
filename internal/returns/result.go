// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

// IOResult describes copy or scan totals reported by a completed operation.
type IOResult interface {
	// Total returns the total number of filesystem entries observed.
	Total() int64
	// Files returns how many regular files were processed.
	Files() int64
	// Links returns how many symbolic links were processed.
	Links() int64
	// Directories returns how many directories were processed.
	Directories() int64

	// SentBytes returns the number of payload bytes written or transferred.
	SentBytes() int64
}
