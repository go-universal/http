package limiter

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// option holds the configuration options for Rate Limiter middleware.
type option struct {
	key      string
	attempts uint
	ttl      time.Duration
	skipFail bool
	fail     func(time.Duration) fiber.Handler
	next     func(*fiber.Ctx) bool
	keys     func(*fiber.Ctx) []string
}

// Option defines a function type for configuring Rate Limiter Option.
type Option func(*option)

// WithMaxAttempts sets the maximum number of attempts allowed.
func WithMaxAttempts(attempts uint) Option {
	return func(o *option) {
		if attempts > 0 {
			o.attempts = attempts
		}
	}
}

// WithTTl sets the time-to-live for the rate limiter.
func WithTTl(ttl time.Duration) Option {
	return func(o *option) {
		if ttl > 0 {
			o.ttl = ttl
		}
	}
}

// WithSkipFail sets the option to skip limiter if request has error.
func WithSkipFail(skipFail bool) Option {
	return func(o *option) {
		o.skipFail = skipFail
	}
}

// WithFail sets a custom failure handler for Rate Limiter validation.
func WithFail(handler func(until time.Duration) fiber.Handler) Option {
	return func(o *option) {
		o.fail = handler
	}
}

// WithNext sets a custom function to skip Rate Limiter validation for certain requests.
func WithNext(handler func(*fiber.Ctx) bool) Option {
	return func(o *option) {
		o.next = handler
	}
}

// WithKeys sets a custom function to generate extra keys based on the request.
func WithKeys(handler func(*fiber.Ctx) []string) Option {
	return func(o *option) {
		o.keys = handler
	}
}
