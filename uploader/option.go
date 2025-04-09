package uploader

import (
	"strings"

	"github.com/go-universal/cache"
)

// option holds configuration settings for the uploader.
type option struct {
	queue    cache.Queue
	numbered bool
	prefix   string
}

// Option defines a function type for modifying uploader option.
type Option func(*option)

// WithQueue sets the queue for managing files that failed to delete.
// Files in the queue must be deleted manually later.
func WithQueue(queue cache.Queue) Option {
	return func(o *option) {
		o.queue = queue
	}
}

// WithNumbered enables numeric file naming.
func WithNumbered() Option {
	return func(o *option) {
		o.numbered = true
	}
}

// WithTimestamped enables timestamp-based file naming.
func WithTimestamped() Option {
	return func(o *option) {
		o.numbered = false
	}
}

// WithPrefix sets a path prefix to exclude from the file URL.
func WithPrefix(prefix string) Option {
	prefix = strings.TrimSpace(prefix)
	return func(o *option) {
		o.prefix = strings.TrimSpace(prefix)
	}
}
